# Documentation Structure

This document explains the organization of SSM Access Manager documentation.

## Root Directory

### Essential Files

- **[README.md](README.md)** - Project overview, features, architecture
- **[QUICKSTART.md](QUICKSTART.md)** - Complete setup and deployment guide (~1 hour)

These are the only documentation files in the root. Everything else is in `docs/`.

## docs/ Directory

All detailed documentation lives here. See **[docs/README.md](docs/README.md)** for the complete index.

### Core Documentation Files

- **[docs/README.md](docs/README.md)** - Documentation index and quick reference
- **[docs/USER_GUIDE.md](docs/USER_GUIDE.md)** - For end users requesting access
- **[docs/ADMIN_GUIDE.md](docs/ADMIN_GUIDE.md)** - Complete administrator guide
- **[docs/OPERATIONS.md](docs/OPERATIONS.md)** - Day-to-day operations and monitoring

## File Organization

```
.
├── README.md                    # Project overview
├── QUICKSTART.md               # Complete setup and deployment guide
├── DOCUMENTATION_STRUCTURE.md  # This file
│
├── docs/
│   ├── README.md               # Documentation index
│   ├── USER_GUIDE.md           # For end users
│   ├── ADMIN_GUIDE.md          # For administrators
│   └── OPERATIONS.md           # For operations teams
│
└── scripts/
    ├── add-user.sh             # Add administrators
    ├── list-users.sh           # List users
    ├── build.sh                # Build Lambdas
    ├── deploy.sh               # Deploy infrastructure
    ├── init-db.sh              # Initialize database
    └── verify-slack-token.sh   # Verify Slack configuration
```

## Documentation Principles

### 1. Keep Root Clean
- Only README.md and QUICKSTART.md in root
- Everything else in docs/

### 2. Single Source of Truth
- Each topic has one primary document
- Other documents link to it
- No duplicate content

### 3. Audience-Focused
- USER_GUIDE.md - For end users
- ADMIN_GUIDE.md - For administrators
- OPERATIONS.md - For operations teams
- Clear navigation in docs/README.md

### 4. Task-Oriented
- Organized by what users want to accomplish
- Step-by-step instructions
- Examples for every feature

### 5. Comprehensive Guides
- QUICKSTART.md includes all deployment and setup
- ADMIN_GUIDE.md includes all admin operations
- No need to jump between multiple files

## Quick Reference

### I want to...

**Deploy the system**
→ [QUICKSTART.md](QUICKSTART.md)

**Request access**
→ [docs/USER_GUIDE.md](docs/USER_GUIDE.md)

**Manage approval groups**
→ [docs/ADMIN_GUIDE.md](docs/ADMIN_GUIDE.md#managing-approval-groups)

**Add administrators**
→ [docs/ADMIN_GUIDE.md](docs/ADMIN_GUIDE.md#managing-administrators)

**Add AWS accounts**
→ [docs/ADMIN_GUIDE.md](docs/ADMIN_GUIDE.md#managing-aws-accounts)

**Troubleshoot issues**
→ [docs/OPERATIONS.md](docs/OPERATIONS.md#troubleshooting) or [docs/ADMIN_GUIDE.md](docs/ADMIN_GUIDE.md#troubleshooting)

**Monitor the system**
→ [docs/OPERATIONS.md](docs/OPERATIONS.md)

**See all documentation**
→ [docs/README.md](docs/README.md)

## Documentation Workflow

### When Adding New Features

1. **Update QUICKSTART.md** - If setup/deployment changes
2. **Update ADMIN_GUIDE.md** - For admin features
3. **Update USER_GUIDE.md** - For user-facing features
4. **Update OPERATIONS.md** - For operational changes
5. **Update docs/README.md** - Add to index
6. **Update README.md** - Update overview if major feature

### Example: Adding New Admin Command

1. ✅ Add command to ADMIN_GUIDE.md with examples
2. ✅ Update command reference section
3. ✅ Add troubleshooting if needed
4. ✅ Update docs/README.md quick reference
5. ✅ Update README.md if it's a major feature

## Maintenance

### Regular Updates

- Review QUICKSTART.md quarterly for accuracy
- Update docs/ when features change
- Keep examples current with latest versions
- Add troubleshooting sections based on issues

### Version Control

- Document breaking changes in docs/README.md
- Update version history in README.md
- Maintain upgrade guides in QUICKSTART.md

