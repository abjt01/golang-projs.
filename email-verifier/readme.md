# Go CLI tool that checks email-related DNS records for a given domain.

## the program prints a CSV-style result with the following fields: [domain,hasMX,hasSPF,spfRecord,hasDMARC,dmarcRecord]

1. MX Record (hasMX) :=
- Checks if the domain has mail servers
- Required to receive email
- Uses DNS MX lookup
If hasMX = false, the domain cannot receive email

2. SPF Record (hasSPF) :=
- SPF (Sender Policy Framework) defines who can send email for the domain
- Stored as a DNS TXT record starting with v=spf1
If missing, emails are more likely to be marked as spam

3. DMARC Record (hasDMARC) :=
- DMARC defines what to do if SPF/DKIM fails
- Stored as a DNS TXT record at _dmarc.domain.com
This helps prevent email spoofing and phishing

--
### how it works internally?
- Uses Go’s standard library only
- Performs real DNS lookups over the internet
- No external APIs
- No regex or string guessing

- Key functions used:
net.LookupMX()
net.LookupTXT()
- --

#### the tool assumes -
- Input is already a domain
- The goal is DNS configuration verification, not email format checking

#### why no external dependencies?
- All logic uses Go’s built-in packages
- No third-party libraries required
- go.mod exists only for good Go practice
-
- helps check email deliverability readiness
- useful for onboarding new domains
- lightweight alternative to online tools
- good learning project for DNS + Go networking
--
-how DNS works at a basic level
-how email authentication is enforced
-how Go performs network lookups
-how to build simple CLI tools in Go
