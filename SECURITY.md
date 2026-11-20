# Security policy

Report vulnerabilities privately to the operator using the contact address
listed at https://hyspayre.com/rechtliches.html. Please do not publish working
exploits or player records.

Never publish: private publisher keys, Supabase service-role keys, email SMTP
passwords, player access/refresh tokens, or Windows Authenticode private keys.
If exposed, rotate them immediately and invalidate affected sessions.

The catalog's Ed25519 public key is *not* a secret. This desktop application
verifies catalog signatures and SHA-256 before applying updates. The authenticator
uses the project's public publishable key; all user-level actions use Supabase
Auth and server-authorized SQL RPCs. Do not weaken server-side access policies.

Code signing for Windows (Authenticode) is distinct from signing the Hyspayre
update catalog. Neither should be bypassed by disabling Windows SmartScreen.
