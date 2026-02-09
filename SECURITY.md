# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.0.1   | :white_check_mark: |

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, please report them via one of the following methods:

- **Email**: security@masterfabric.co
- **GitHub Security Advisory**: Use the [Security tab](https://github.com/masterfabric-go/masterfabric-go/security/advisories/new) to create a private security advisory

### What to Include

When reporting a security vulnerability, please include:

- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

### Response Timeline

- **Initial Response**: Within 48 hours
- **Status Update**: Within 7 days
- **Fix Timeline**: Depends on severity (typically 30-90 days)

### Security Best Practices

When using masterfabric-go in production:

- ✅ Change default `JWT_SECRET` to a strong, random value
- ✅ Use SSL/TLS for database connections (`DB_SSLMODE=require`)
- ✅ Enable rate limiting for production workloads
- ✅ Regularly update dependencies (`go get -u ./...`)
- ✅ Review and rotate API keys regularly
- ✅ Monitor audit logs for suspicious activity
- ✅ Use environment variables for sensitive configuration
- ✅ Keep Docker images updated

### Known Security Considerations

- **JWT Tokens**: Default 24h expiry; adjust `JWT_EXPIRATION_HOURS` as needed
- **Password Hashing**: Uses bcrypt with cost factor 10
- **API Keys**: Store securely; secrets are only shown once on creation
- **Rate Limiting**: Configure per-endpoint via policies
- **CORS**: Currently allows all origins (`*`); restrict in production

### Security Updates

Security updates will be:
- Documented in CHANGELOG.md
- Tagged with security labels
- Released as patch versions

Thank you for helping keep masterfabric-go secure! 🔒
