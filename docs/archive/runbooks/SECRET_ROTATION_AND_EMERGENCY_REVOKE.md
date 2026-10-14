# Secret rotation and emergency revoke runbook

This runbook is the operator contract; it intentionally does not contain live
secret values.

## Emergency session revoke

1. Identify the affected user or incident scope through the incident channel.
2. Call the authenticated admin operation that invokes
   `RevokeAllUserSessions(userID)`; never delete session rows by hand during an
   active incident unless the approved database break-glass procedure is in
   effect.
3. Confirm the session store reports no active sessions for that user and
   record the opaque incident/request ID in the audit trail.
4. Rotate the suspected credential in the deployment secret manager and
   restart/reload according to the deployment procedure.

## MFA master-key rotation

MFA ciphertext is bound to the configured master key. Do not overwrite the key
and assume existing factors remain readable. Stage a new deployment with a
planned migration procedure, re-enroll affected MFA factors, and retain the
old key only in the approved secure recovery channel until verification is
complete. A real key ceremony and retention decision are external production
gates.

## Verification and rollback

- Never place raw keys, tokens, cookies, or recovery codes in logs, exports, or
  tickets.
- Verify login, logout, MFA verification, and recovery flows after rotation.
- If verification fails, stop rollout and use the deployment rollback path;
  do not print key material to diagnose the failure.
