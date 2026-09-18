# API input limits

All body/JSON/query bindings in REST handlers use the shared validation helpers.
JSON field names are also used for URL-encoded and multipart form fields. Existing
endpoint requirements and business rules still apply after these upper bounds.

| Input | Limit |
| --- | --- |
| Project and milestone title | 50 Unicode characters; supplied update titles cannot be blank |
| Project short description | 40 characters, matching the database field |
| Other descriptions, messages, questions, answers, summaries | 5,000 characters |
| General text, including story/update bodies | 10,000 characters |
| Names and search text | 100 characters |
| Notes, reasons, biography, address | 2,000 characters |
| URLs | 2,048 characters; HTTP/HTTPS, host required, no embedded credentials |
| Passwords | 72 bytes, matching bcrypt's input limit |
| Money inputs | Finite, nonnegative, at most 1,000,000,000; endpoint limits may be stricter |
| Percentages | 0–100 |
| Fundraising duration | At most 60 days; project service requires at least 1 |
| Project duration | At most 48 months; project service requires at least 1 |
| Milestone duration | At most 365 days |
| Milestones | At most 4 per project; phases 1–4, unique per project |
| Pictures / videos | At most 5 of each, up to 10 combined |
| Raw documents | At most 5, also subject to the combined 10-file cap |
| External milestone links | At most 5 |
| Checklist items | At most 50 |

Text rejects null characters. Numeric validation rejects NaN and either infinity,
including form/query inputs. Signed input integers are capped at 2,147,483,647;
unsigned IDs must fit a signed 64-bit database integer. Optional fields remain
optional; this is validation rather than silent truncation or rewriting.

## Uploads and saved media

`POST /upload` accepts `file` for one file or repeated `files` for a batch. It checks
the entire batch before contacting storage. File content must match the filename
extension. Images/documents are limited to 5 MiB each, videos to 50 MiB. The server's
existing 55 MiB total request limit still applies, including multipart overhead;
larger collections must be uploaded in smaller batches.

Saved project media has the same 5-picture/5-video limits across requests. Each
file has exactly one media type. Attachment batches are saved in one transaction.
Project row locks serialize count checks and writes across server instances;
changing a saved file's type also checks the resulting counts. Milestone creation
uses the same lock to enforce four milestones and allocate an unused phase.

Milestone URLs and submission attachments allow 5 pictures plus 5 videos. Their
types are inferred from recognized URL extensions or Cloudinary resource paths;
unknown document URLs count as raw. Remote URL contents are not downloaded to
verify their type. The upload endpoint checks actual file bytes.

## Verification

Regression tests cover Unicode title boundaries, non-finite numbers, form field
decoding, mixed upload batches, excess files, unsupported/mismatched file content,
image size boundaries, media type changes, and rejection before writes.

The initial suite reproduced form fields being ignored, rejection of a valid
5-picture/5-video batch, and outdated success fixtures missing required inputs.
Fixtures now supply the required data and mock identity-verification HTTP calls.

Database locking still needs a live PostgreSQL concurrency test. Docker's engine
was unavailable in the development environment. Existing stored data is not
rewritten by this change.

## Audit fixes

- Password creation/reset policy counts at least 8 Unicode code points and allows
  at most 72 UTF-8 bytes. Password content is not truncated.
- Complaint subject/body and resolution notes must satisfy their minimum lengths
  after trimming surrounding whitespace. Evidence uses the shared HTTP/HTTPS URL
  checks and 2,048-character maximum.
- The owner milestone edit endpoint rejects any supplied `status` with HTTP 400.
  Clients must use the submission, approval, voting and payment workflows for
  transitions. Creating a milestone preserves its supplied title, description,
  duration, acceptance criteria and media; its initial status remains `draft`.
  Phase and sort order are allocated by the server.
- Empty or whitespace-only chat messages return HTTP 400.
