-- 017_mail_templates_in_code.sql
-- Mail templates and SMTP config now live in application code/env vars.

DROP TABLE IF EXISTS tmail_templates;

DELETE FROM tconfig
WHERE key IN ('smtp_host', 'smtp_port', 'smtp_user', 'smtp_password', 'smtp_from');
