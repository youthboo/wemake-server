# Wemake Server API Spec

Base URL

```txt
https://wemake-server.onrender.com/api/v1
```

Health Check

```txt
GET https://wemake-server.onrender.com/health
```

Expected response

```json
{
  "status": "ok"
}
```

## Authentication

Recommended auth flow for frontend:

1. Call `POST /auth/login`
2. Save the returned JWT token
3. Send the token on subsequent requests:

```http
Authorization: Bearer <JWT_TOKEN>
Content-Type: application/json
```

The server also supports `X-User-ID` as a fallback for internal/testing flows, but frontend should prefer `Bearer` token.

### `POST /auth/register`

Register customer account:

```json
{
  "role": "CT",
  "email": "customer@example.com",
  "phone": "0812345678",
  "password": "P@ssw0rd123",
  "first_name": "Somchai",
  "last_name": "Jaidee"
}
```

Register factory account:

```json
{
  "role": "FT",
  "email": "factory@example.com",
  "phone": "0899999999",
  "password": "P@ssw0rd123",
  "factory_name": "My Factory",
  "factory_type_id": 1,
  "tax_id": "0105555xxxxxx"
}
```

Returns:

```json
{
  "token": "JWT_TOKEN",
  "user": {}
}
```

### `POST /auth/login`

Request

```json
{
  "email": "user@example.com",
  "password": "your-password"
}
```

Response

```json
{
  "token": "JWT_TOKEN",
  "user": {
    "user_id": 1,
    "role": "CT",
    "email": "user@example.com",
    "phone": "0812345678",
    "is_active": true,
    "created_at": "2026-03-22T10:00:00Z",
    "updated_at": "2026-03-22T10:00:00Z"
  }
}
```

### `POST /auth/forgot-password`

Request

```json
{
  "email": "user@example.com"
}
```

Response

```json
{
  "message": "if the account exists, reset instructions have been generated",
  "reset_token": "optional-for-testing"
}
```

### `POST /auth/reset-password`

Request

```json
{
  "token": "reset-token",
  "new_password": "N3wP@ssword123"
}
```

## Frontend-Ready Endpoints

These are the best endpoints for the current client to integrate with now.

### `GET /frontend/mock-data`

Best starting point for current client integration.  
Returns a payload shaped close to the current `mockData.ts`.

Top-level keys:

```json
{
  "currentUser": {},
  "categories": [],
  "factories": [],
  "factoryProfiles": [],
  "factoryReviews": [],
  "ideaArticles": [],
  "factoryShowcases": [],
  "rfqs": [],
  "orders": [],
  "conversations": [],
  "notifications": []
}
```

Recommended FE usage:

1. Login
2. Call `GET /frontend/mock-data`
3. Map response to replace the current local mock bundle

### `GET /frontend/bootstrap`

Aggregated dashboard payload:

```json
{
  "currentUser": {},
  "categories": [],
  "factories": [],
  "rfqs": [],
  "orders": [],
  "threads": []
}
```

### `GET /frontend/me`

Returns current user summary:

```json
{
  "id": 1,
  "role": "CT",
  "name": "Somchai Jaidee",
  "company": "Wemake Member",
  "email": "user@example.com",
  "phone": "0812345678",
  "avatar": "",
  "walletBalance": 0,
  "pendingBalance": 0,
  "memberSince": "2026"
}
```

### `GET /frontend/factories`

Returns frontend-friendly factory cards.

### `GET /frontend/factories/:factory_id`

Returns one factory detail:

```json
{
  "factory": {},
  "profile": {},
  "reviews": [],
  "products": [],
  "promotions": [],
  "ideas": []
}
```

### `GET /frontend/rfqs/:rfq_id`

Returns one RFQ detail with offers and images.

### `GET /frontend/orders/:order_id`

Returns one order detail with timeline.

### `GET /frontend/messages/threads`

Returns frontend-friendly conversation/thread list.

## Catalog Endpoints

### `GET /categories`

Returns marketplace categories.

### `GET /units`

Returns marketplace units.

## Factory CRUD Endpoints

### `POST /factories/`

Request

```json
{
  "name": "Factory Name",
  "email": "factory@example.com",
  "phone": "0881234567",
  "address": "123 Factory Road",
  "description": "Factory description"
}
```

### `GET /factories/`

List all factories.

### `GET /factories/:id`

Get one factory.

### `PATCH /factories/:id`

Update factory.

### `DELETE /factories/:id`

Delete factory.

## Address Endpoints

Auth required.

### `GET /addresses/`

List current user's addresses.

### `POST /addresses/`

Request

```json
{
  "address_type": "C",
  "address_detail": "123 ถนนสุขุมวิท",
  "sub_district_id": 100101,
  "district_id": 1001,
  "province_id": 1,
  "zip_code": "10110",
  "is_default": true
}
```

### `PATCH /addresses/:address_id`

Partial update address fields.

## RFQ Endpoints

Auth required.

### `POST /rfqs/`

Request

```json
{
  "category_id": 1,
  "title": "New RFQ",
  "quantity": 1000,
  "unit_id": 1,
  "budget_per_piece": 42.5,
  "details": "Project details",
  "address_id": 1
}
```

### `GET /rfqs/`

Query param supported:

- `status`

### `GET /rfqs/:rfq_id`

Returns:

```json
{
  "rfq": {},
  "images": []
}
```

### `POST /rfqs/:rfq_id/images`

```json
{
  "image_url": "https://example.com/image.png"
}
```

### `PATCH /rfqs/:rfq_id/cancel`

Cancel RFQ.

### `POST /rfqs/:rfq_id/quotations`

Create quotation under RFQ.

### `GET /rfqs/:rfq_id/quotations`

List quotations under RFQ.

## Quotation Endpoints

### `GET /quotations/:quotation_id`

Get quotation detail.

### `PATCH /quotations/:quotation_id/status`

```json
{
  "status": "AC"
}
```

Allowed values:

- `PD`
- `AC`
- `RJ`

## Wallet Endpoints

Auth required.

### `GET /wallets/me`

Get current user's wallet.

## Order Endpoints

Auth required.

### `POST /orders/`

```json
{
  "quote_id": 1
}
```

### `GET /orders/`

Query param supported:

- `status`

### `GET /orders/:order_id`

Get one order.

### `PATCH /orders/:order_id/status`

```json
{
  "status": "PR"
}
```

Allowed values:

- `PR`
- `QC`
- `SH`
- `CP`

### `POST /orders/:order_id/production-updates`

```json
{
  "step_id": 1,
  "description": "Production update",
  "image_url": "https://example.com/image.png"
}
```

### `GET /orders/:order_id/production-updates`

List production updates.

## Production Update Endpoints

### `PATCH /production-updates/:update_id`

```json
{
  "description": "Updated description",
  "image_url": "https://example.com/new-image.png"
}
```

## Message Endpoints

Auth required.

### `POST /messages/`

```json
{
  "reference_type": "RFQ",
  "reference_id": "1",
  "receiver_id": 2,
  "content": "Hello",
  "attachment_url": "",
  "conv_id": 1,
  "message_type": "TX",
  "quote_data": "{\"price\": 65000, \"lead_time\": 21}"
}
```
*Note: `conv_id`, `message_type` (e.g., TX or QT), and `quote_data` are optional but strongly recommended for conversation threading and rich quoting.*

### `GET /messages/`

Supports fetching by `conv_id` natively, or fallback to references.

Examples:

```txt
GET /messages/?conv_id=1
GET /messages/?reference_type=RFQ&reference_id=1
```

### `GET /messages/threads`

List message threads for current user.

## Transaction Endpoints

### `POST /transactions/`

Create transaction.

### `GET /transactions/`

List transactions.

### `PATCH /transactions/:tx_id/status`

Patch transaction status.

## Master Data Endpoints

### `GET /master/provinces`

### `GET /master/districts?province_id=1`

### `GET /master/sub-districts?district_id=1001`

### `GET /master/factory-types`

### `GET /master/product-categories`

### `GET /master/production-steps`

### `GET /master/units`

### `GET /master/shipping-methods`

## Frontend Handoff Summary

Share this with frontend:

```txt
API_BASE_URL=https://wemake-server.onrender.com/api/v1

Health:
GET /health

Login:
POST /auth/login

Use after login:
Authorization: Bearer <token>

Best first endpoint:
GET /frontend/mock-data
```

Recommended frontend rollout:

1. Integrate login
2. Integrate `GET /frontend/mock-data`
3. Replace local mock data usage
4. Migrate to granular `/frontend/*` endpoints page by page

## Media Endpoints

### `POST /media/upload`

Upload a file (image, document):

```http
Content-Type: multipart/form-data
Body: file=<file_binary>
```

Returns:
```json
{
  "url": "http://SERVER_URL/uploads/xyz123.jpg",
  "file_name": "xyz123.jpg",
  "size": 102400
}
```

## New Frontend Endpoints

### `GET /frontend/products`
Query params: `limit` (default 8)

### `GET /frontend/promotions`
Query params: `limit` (default 4)

### `GET /frontend/promo-codes`

### `GET /frontend/explore`
Aggregated endpoint returning Products, Promotions, PromoCodes, Factories, Categories, IdeaArticles.

## Factory Reviews Endpoints

Auth required for POST.

### `GET /factories/:factory_id/reviews`

List reviews for a factory.

### `POST /factories/:factory_id/reviews`

Create a review:

```json
{
  "rating": 5,
  "comment": "Great factory!"
}
```

## Factory Certificates Endpoints

Auth required for POST (must be the factory owner).

### `GET /factories/:factory_id/certificates`

List certificates for a factory.

### `POST /factories/:factory_id/certificates`

```json
{
  "cert_id": 1,
  "document_url": "http://SERVER_URL/uploads/cert.pdf",
  "expire_date": "2027-12-31",
  "cert_number": "TH12/3456"
}
```

## Conversations Endpoints

Auth required.

### `GET /conversations`

List current user's chat conversations.

### `GET /conversations/:conv_id`

Get single conversation details.

### `POST /conversations`

Create a new conversation:

```json
{
  "customer_id": 1,
  "factory_id": 2
}
```

## Notifications Endpoints

Auth required.

### `GET /notifications`

List current user's notifications.

### `PATCH /notifications/:noti_id/read`

Mark a notification as read.

## Showcases Endpoints

### `GET /showcases`

List showcases (newest first). Optional filter:

| Query | Value | Meaning |
|-------|-------|---------|
| `type` | `PD` | Products only |
| `type` | `PM` | Promotions only |
| `type` | `ID` | Ideas only |
| *(omit)* | — | All content types |

Examples: `GET /api/v1/showcases`, `GET /api/v1/showcases?type=PD`

Invalid `type` returns `400` with an error message.

### `POST /showcases`

Auth required (for factory).

```json
{
  "content_type": "PD",
  "title": "New Packaging Product",
  "excerpt": "Great for food",
  "image_url": "http://SERVER_URL/img.jpg",
  "category_id": 1,
  "min_order": 1000,
  "lead_time_days": 14
}
```

## Promo Slides Endpoints

### `GET /promo-slides`

List active promotional banners/slides.

## Favorites Endpoints

Auth required.

### `GET /favorites`

List user's favorite showcases.

### `POST /favorites`

Add a showcase to favorites:

```json
{
  "showcase_id": 1
}
```

### `DELETE /favorites/:showcase_id`

Remove from favorites.
