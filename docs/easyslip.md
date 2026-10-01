# EasySlip profit transfer verification

Pioneer profit submissions and admin investor payout confirmations now require a
slip image uploaded through Flyup. The backend calls the
[EasySlip v2 bank verification API](https://document.easyslip.com/th/v2/verify/bank/url).
The API key stays on the backend.

## Configuration

Set in the backend `.env` or deployment environment:

```dotenv
EASYSLIP_API_KEY=<API key for the flyup branch>
PLATFORM_BANK_CODE=004
PLATFORM_ACCOUNT_NUMBER=<Flyup receiving account>
```

`004` is KBANK. The local receiving account is configured from the account supplied
by the owner. Keep the frontend's `VITE_PLATFORM_BANK_NAME`,
`VITE_PLATFORM_ACCOUNT_NAME`, and `VITE_PLATFORM_ACCOUNT_NUMBER` consistent with
this account. Restart the backend and rebuild/restart the frontend after changing
environment settings.

Fund/activate the EasySlip branch and configure any required IP whitelist for the
backend's outbound address. For masked slip account numbers, register the **Flyup
receiving account and investor receiving accounts** in EasySlip and link them to
the branch used by the API key. See
[bank account registration](https://document.easyslip.com/th/v2/bank-accounts/).
The verifier requires either an exact provider-matched account and bank, or an
unmasked full BANKAC account number and matching bank. Names or masked suffixes
alone cannot authorize a payout.

## Behavior

- Pioneer: choose project/quarter, enter transferred amount, upload slip, submit.
  Only a verified transfer into the configured Flyup account creates the pool and
  its investor payouts. The amount is compared to the submitted amount; this does
  not audit the pioneer's profit calculation.
- Admin: transfer to the investor's default receiving account, attach slip, confirm.
  The expected amount comes from the stored investor payout, not the browser.
- Admin pool creation via API also requires a verified incoming slip; it cannot
  bypass the Pioneer checks by submitting only a manual reference.
- References come from EasySlip's `rawSlip.transRef`, regardless of a client-supplied
  `transfer_ref`. The old input is accepted for compatibility but is not trusted.
- Slip URL, provider transfer time, verification time and reference are saved.
  The shared `verified_slips` unique index on sender bank and bank transfer reference
  prevents reuse across incoming pools and outgoing payouts, including concurrent
  requests. Consumption and business records commit in a single DB transaction.
- Provider `checkDuplicate` is false intentionally: local transactional consumption
  permits retry after a failed DB write without permanently burning a slip at the
  provider. This prevents duplicate use inside Flyup, not across unrelated apps.
- Missing credentials, no quota, unavailable provider, pending bank data, wrong
  amount/account, or inability to confirm the full account all block confirmation.
  Stripe test/live mode does not change this verification into a simulated payment.
- Database startup AutoMigrate adds the verification table and slip fields. Existing
  manually confirmed records retain their old status and have no verification badge.

## Request fields

`POST /pioneer/profit-pools/:projectId`:

```json
{"quarter_no":1,"total_amount":1500.50,"slip_image":"https://res.cloudinary.com/<cloud>/image/upload/<slip>"}
```

`PATCH /admin/profit-pools/:id/payouts/:payoutId/confirm`:

```json
{"slip_image":"https://res.cloudinary.com/<cloud>/image/upload/<slip>","note":""}
```

Images must be JPEG, PNG, GIF or WebP, at most 4 MB, with a clear QR code. Only
HTTPS image upload URLs in the configured Cloudinary cloud are accepted.

## Validation performed

Client tests use a mock HTTP provider for wrong amounts/banks/accounts, masked
accounts, duplicate flags, invalid references/dates, pending slips, absent keys,
and invalid URLs. Repository tests exercise cross-flow duplicate consumption,
rollback/retry and conditional confirmation. Service tests exercise both incoming
paths and outgoing payouts with a fake verifier. No real slip or paid EasySlip API
request is made by these tests.

For a live test, configure the key/branch, use a genuine transfer, then verify the
stored reference and `slip_verified_at`. Reusing that same slip in a second business
record must fail. Rejected requests must leave the payout pending.
