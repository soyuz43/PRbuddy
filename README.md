```markdown
# PRBuddy-Go :robot:

**AI-Powered Pull Request Automation Suite**  
*Local Execution · Git Integration · Developer-Centric Workflows*

---

## :sparkles: Features

1. **Automated PR Draft Generation**
   - Post-commit hook integration
   - Context-aware LLM-powered descriptions
   - Multi-format diff analysis (staged/unstaged/untracked)

2. **Intelligent Change Analysis**
   ```bash
   prbuddy-go what
   ```
   - Natural language summaries of recent changes
   - File-by-file breakdown of modifications
   - Temporal context preservation

3. **Developer Assistant Integration**
   ```bash
   prbuddy-go quickassist "Explain this Go interface"
   ```
   - Code-aware Q&A system
   - Ephemeral conversation contexts
   - Cross-file understanding

4. **Extension Ecosystem**
   - VS Code integration
   - Real-time collaboration
   - Contextual draft editing

5. **Privacy-First Architecture**
   - 100% local execution
   - Zero cloud dependencies
   - End-to-end in-process data handling

---

## :computer: Installation

### Prerequisites
- [Go 1.20+](https://go.dev/dl/)
- [Ollama](https://ollama.ai/) (Local LLM Server)
- [Git](https://git-scm.com/)

### Quick Start
```bash
# Install binary
go install github.com/soyuz43/prbuddy-go@latest

# Initialize in your repo
cd your-project
prbuddy-go init
```

### Model Configuration
```bash
# Set default LLM (Ollama required)
export PRBUDDY_LLM_MODEL="codellama:7b"

# Verify model availability
curl http://localhost:11434/api/ps
```

---

## :rocket: Usage

### Workflow Automation
```bash
# After commit hook triggers automatic PR draft
git commit -m "feat: add authentication middleware"

# Manual PR generation
prbuddy-go post-commit --extension-active
```

### Change Intelligence
```bash
# Get detailed change summary
prbuddy-go what

# Sample output:
"""
**Modified Files:**
1. `pkg/auth/jwt.go`
   - Added JWT validation layer
   - Implemented token refresh logic
2. `cmd/server/main.go`
   - Integrated auth middleware
   - Updated route configurations
"""
```

### AI-Powered Assistance
```bash
# Code-specific questions
prbuddy-go quickassist "How can I optimize this Goroutine pool?"

# Context-aware refinement
prbuddy-go quickassist "Rephrase this error handling for clarity"
```

---

## :gear: Configuration

### Environment Variables
| Variable | Default | Description |
|----------|---------|-------------|
| `PRBUDDY_LLM_MODEL` | `hermes3` | Local LLM model name |
| `PRBUDDY_LLM_ENDPOINT` | `http://localhost:11434` | Ollama API location |
| `PRBUDDY_MAX_CONTEXT` | `8192` | LLM context window size |

### Extension Endpoints
```javascript
// VS Code extension communication
const endpoints = {
  DRAFTS: '/extension/drafts',
  QUICK_ASSIST: '/extension/quick-assist',
  MODEL_MGMT: '/extension/models'
};
```

---

## :test_tube: Troubleshooting

**Common Issues**  
```bash
# Reset installation
prbuddy-go remove && prbuddy-go init

# Diagnostic commands
prbuddy-go what --debug
prbuddy-go post-commit --non-interactive -v
```

**Port Conflicts**  
```bash
# Check active ports
lsof -i :11434

# Custom port configuration
export PRBUDDY_LLM_ENDPOINT="http://localhost:22834"
```

---

## :handshake: Contributing

1. Fork repository
2. Create feature branch
3. Submit PR with:
   - Test coverage
   - Updated documentation
   - Changelog entry

```bash
# Development setup
make setup
make test
make integration
```

---

## :shield: License

Apache 2.0 © [Your Name]  
*Preserving developer agency through ethical AI tooling*
```
