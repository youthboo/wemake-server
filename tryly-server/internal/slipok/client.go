// Package slipok integrates the SlipOK payment-slip verification service.
//
// SlipOK only "reads the slip and tells the truth from the bank" — it does not
// decide whether a transaction is valid for us. We take the returned data
// (amount, receiver, timestamp, reference) and compare it ourselves against
// what the order expects. See doc/slip-verification-flow-spec.md.
package slipok

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// Status groups every SlipOK outcome into 4 buckets (never just pass/fail).
type Status string

const (
	// StatusOK — slip read successfully, real transaction data returned.
	StatusOK Status = "ok"
	// StatusInvalid — not a slip / unreadable QR / expired / no real txn / duplicate.
	// NOT a failure of the transaction: ask the user to attach a new image.
	StatusInvalid Status = "invalid"
	// StatusRetry — source bank delays data availability (BBL/SCB) or temporary
	// bank outage. Ask the user to wait and resubmit — NOT a rejection.
	StatusRetry Status = "retry"
	// StatusUnavailable — OUR verifier has a problem (bad key / quota / network).
	// Never reject the user's transaction for our own problem → manual fallback.
	StatusUnavailable Status = "unavailable"
)

// Result is the normalized outcome of a verification call.
type Result struct {
	Status       Status
	Code         int
	Message      string
	DelayMinutes int

	// Parsed transaction data (present when Status == StatusOK)
	TransRef            string
	Amount              float64
	ReceiverAccount     string // masked, e.g. "xxx-x-x3109-x"
	ReceiverProxy       string // masked, e.g. "086xxx0000"
	ReceiverName        string
	ReceiverDisplayName string
	TransferredAt       time.Time

	Raw json.RawMessage // full API response for audit/debug
}

// Client talks to SlipOK. Enabled reports whether credentials are configured.
type Client struct {
	branchID string
	apiKey   string
	http     *http.Client
}

func NewClient(branchID, apiKey string) *Client {
	return &Client{
		branchID: strings.TrimSpace(branchID),
		apiKey:   strings.TrimSpace(apiKey),
		http:     &http.Client{Timeout: 20 * time.Second},
	}
}

// Enabled is true only when both credentials are present.
func (c *Client) Enabled() bool {
	return c != nil && c.branchID != "" && c.apiKey != ""
}

type apiResponse struct {
	Success bool            `json:"success"`
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type apiData struct {
	Success        bool    `json:"success"`
	TransRef       string  `json:"transRef"`
	TransDate      string  `json:"transDate"`
	TransTime      string  `json:"transTime"`
	TransTimestamp string  `json:"transTimestamp"`
	Amount         float64 `json:"amount"`
	Receiver       struct {
		DisplayName string `json:"displayName"`
		Name        string `json:"name"`
		Proxy       struct {
			Value string `json:"value"`
		} `json:"proxy"`
		Account struct {
			Value string `json:"value"`
		} `json:"account"`
	} `json:"receiver"`
	BankName string `json:"bankName"`
	Delay    int    `json:"delay"`
}

// VerifyByURL sends the slip image URL to SlipOK.
// NOTE: the URL must be PUBLICLY reachable by SlipOK's servers — a localhost /
// private URL makes SlipOK fail with "รูปภาพไม่ถูกต้อง". Prefer VerifyByFile.
// log is left false: SlipOK just reads the slip; WE do amount/receiver/dedup checks.
func (c *Client) VerifyByURL(imageURL string) *Result {
	if !c.Enabled() {
		return &Result{Status: StatusUnavailable, Message: "SlipOK not configured"}
	}
	body, _ := json.Marshal(map[string]interface{}{"url": imageURL})
	url := fmt.Sprintf("https://api.slipok.com/api/line/apikey/%s", c.branchID)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return &Result{Status: StatusUnavailable, Message: err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-authorization", c.apiKey)
	return c.do(req)
}

// VerifyByFile uploads the slip image bytes to SlipOK as multipart/form-data.
// Works regardless of where the app is hosted (no public URL needed).
func (c *Client) VerifyByFile(filename string, data []byte) *Result {
	if !c.Enabled() {
		return &Result{Status: StatusUnavailable, Message: "SlipOK not configured"}
	}
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("files", filename)
	if err != nil {
		return &Result{Status: StatusUnavailable, Message: err.Error()}
	}
	if _, err := fw.Write(data); err != nil {
		return &Result{Status: StatusUnavailable, Message: err.Error()}
	}
	if err := w.Close(); err != nil {
		return &Result{Status: StatusUnavailable, Message: err.Error()}
	}
	url := fmt.Sprintf("https://api.slipok.com/api/line/apikey/%s", c.branchID)
	req, err := http.NewRequest(http.MethodPost, url, &buf)
	if err != nil {
		return &Result{Status: StatusUnavailable, Message: err.Error()}
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("x-authorization", c.apiKey)
	return c.do(req)
}

func (c *Client) do(req *http.Request) *Result {
	resp, err := c.http.Do(req)
	if err != nil {
		// network/timeout → our side is unavailable, never blame the user
		return &Result{Status: StatusUnavailable, Message: err.Error()}
	}
	defer resp.Body.Close()

	var ar apiResponse
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&ar); err != nil {
		return &Result{Status: StatusUnavailable, Message: "invalid SlipOK response"}
	}

	res := &Result{Code: ar.Code, Message: ar.Message, Raw: ar.Data}

	if resp.StatusCode == http.StatusOK && ar.Success {
		res.Status = StatusOK
		parseData(ar.Data, res)
		return res
	}

	// Non-200 → classify by error code (see SlipOK API Guide §Error Status Code)
	switch ar.Code {
	case 1009, 1010: // bank temporarily down / bank delay slip
		res.Status = StatusRetry
		var d apiData
		if json.Unmarshal(ar.Data, &d) == nil && d.Delay > 0 {
			res.DelayMinutes = d.Delay
		}
	case 1002, 1003, 1004, 1015: // bad key / expired package / over quota / no package
		res.Status = StatusUnavailable
	default: // 1000,1005,1006,1007,1008,1011,1012,1013,1014 → bad slip / duplicate
		res.Status = StatusInvalid
	}
	return res
}

func parseData(raw json.RawMessage, res *Result) {
	var d apiData
	if len(raw) == 0 || json.Unmarshal(raw, &d) != nil {
		return
	}
	res.TransRef = strings.TrimSpace(d.TransRef)
	res.Amount = d.Amount
	res.ReceiverAccount = strings.TrimSpace(d.Receiver.Account.Value)
	res.ReceiverProxy = strings.TrimSpace(d.Receiver.Proxy.Value)
	res.ReceiverName = strings.TrimSpace(d.Receiver.Name)
	res.ReceiverDisplayName = strings.TrimSpace(d.Receiver.DisplayName)
	res.TransferredAt = parseTransferredAt(d)
}

func parseTransferredAt(d apiData) time.Time {
	if d.TransTimestamp != "" {
		if t, err := time.Parse(time.RFC3339, d.TransTimestamp); err == nil {
			return t
		}
	}
	// Fallback: build from transDate (yyyyMMdd) + transTime (HH:mm:ss), Bangkok tz
	if d.TransDate != "" && d.TransTime != "" {
		loc, err := time.LoadLocation("Asia/Bangkok")
		if err != nil {
			loc = time.FixedZone("ICT", 7*3600)
		}
		if t, err := time.ParseInLocation("20060102 15:04:05", d.TransDate+" "+d.TransTime, loc); err == nil {
			return t
		}
	}
	return time.Time{}
}

var nonAccountChars = regexp.MustCompile(`[^0-9xX]`)

// AccountMatches compares a masked value from the API (e.g. "xxx-x-x3109-x")
// against the full expected account/proxy we control. Right-aligned, position by
// position; 'x' positions are skipped; requires ≥2 visible digits to count as a
// real match (so a fully-masked value never passes for free). See spec §4.
func AccountMatches(maskedAPI, fullExpected string) bool {
	masked := strings.ToLower(nonAccountChars.ReplaceAllString(maskedAPI, ""))
	full := nonAccountChars.ReplaceAllString(fullExpected, "")
	full = strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, full)
	if masked == "" || full == "" {
		return false
	}
	visible := 0
	mi, fi := len(masked)-1, len(full)-1
	for mi >= 0 && fi >= 0 {
		mc := masked[mi]
		if mc != 'x' {
			if mc != full[fi] {
				return false
			}
			visible++
		}
		mi--
		fi--
	}
	return visible >= 2
}
