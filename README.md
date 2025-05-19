# PRBuddy-Go 🚀🤖

> Automate pull request drafting and code reasoning with your Git history – powered by LLMs and Git hooks.

![Go](https://img.shields.io/badge/Go-1.20+-brightgreen)
![License](https://img.shields.io/github/license/soyuz43/prbuddy)
![PRBuddy Status](https://img.shields.io/badge/status-alpha-orange)

---

## ✨ What Is PRBuddy-Go?

PRBuddy-Go is a lightweight CLI assistant that integrates into your Git workflow. It automatically generates pull request drafts after every commit and helps you understand your changes with natural language summaries.

Whether you're working solo or in a team, PRBuddy helps you keep your code explainable and your PRs professional — effortlessly.

---

## 🛠 Features

- 🧠 **LLM-powered PR Drafts**: Hooks into `post-commit` to auto-generate contextual pull request messages.
- 💬 **Quick Assist Chat**: Get fast, contextual help from an LLM in your terminal.
- 🕵️ **"What did I just do?"** summaries with `prbuddy-go what`
- ⚙️ **Optional Git hook installation** during `init`
- 🧼 **Cleanup** with `prbuddy-go remove`

---

## 📦 Installation

### Prerequisites

- Go 1.20 or later
- Git (with a local repo)

### Install

```bash
go install github.com/soyuz43/prbuddy-go@latest
```

> Or clone and build manually:

```bash
git clone https://github.com/soyuz43/PRbuddy.git
cd PRbuddy
go build -o prbuddy-go
```

---

## ⚡ Quick Start

```bash
cd your-project/
prbuddy-go init        # Installs Git hook + .git/pr_buddy_db
git add .
git commit -m "feat: add logging"  # Triggers PR draft generation
```

---

## 🧪 Commands

| Command               | Description                                               |
| --------------------- | --------------------------------------------------------- |
| `init`                | Setup PRBuddy in current repo; installs optional Git hook |
| `post-commit`         | Used internally by the hook to draft PR messages          |
| `what`                | Summarize local changes since last commit                 |
| `quickassist [query]` | Ask the LLM anything, or run interactive CLI chat         |
| `remove`              | Uninstall PRBuddy from the repo                           |

---

## 🧠 How It Works

* Uses **Git hooks** to run logic after commits
* Detects branch, commit, diff context
* Sends data to an **LLM backend** (e.g., OpenAI, local model?)
* Generates structured PR drafts
* Stores metadata in `.git/pr_buddy_db` for traceability

> ✅ You can disable or uninstall anytime using: `prbuddy-go remove`

---

## 🔐 Privacy & Security

PRBuddy reads your local Git data and may transmit code context to an LLM service. Make sure you're comfortable with the models you're using and consider privacy policies if sensitive code is involved.

---

## 🤝 Contributing

This project is in early development. Bug reports, ideas, and PRs are welcome!

---

## 📄 License

MIT © [soyuz43](https://github.com/soyuz43)


