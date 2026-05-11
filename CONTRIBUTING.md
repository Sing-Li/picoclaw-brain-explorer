# Contributing to PicoClaw Brain Explorer

Thank you for your interest in contributing to PicoClaw Brain Explorer! This project is a community-driven effort to create quanlity tooling for PicoClaw development.  We welcome contributions of all kinds: bug fixes, features, documentation, translations, and testing.

The project itself was substantially developed with AI assistance — we embrace this approach and have built our contribution process around it.


---

## Code of Conduct

We are committed to maintaining a welcoming and respectful community. Be kind, constructive, and assume good faith. Harassment or discrimination of any kind will not be tolerated.

---

## Ways to Contribute

- **Bug reports** — Open an issue using the bug report template.
- **Feature requests** — Open an issue using the feature request template; discuss before implementing.
- **Code** — Fix bugs or implement features. See the roadmap to see what we think still needs to be done.
- **Documentation** — Improve READMEs, docs, inline comments, or translations.
- **Testing** — Run it against your own PicoClaw report your results.

For substantial new features, please open an issue first to discuss the design before writing code. This prevents wasted effort and ensures alignment with the project's direction.

---

## AI-Assisted Contributions

PicoClaw was built with substantial AI assistance, and we fully embrace AI-assisted development. However, contributors must understand their responsibilities when using AI tools.

### Disclosure Is Required

Every PR must disclose AI involvement using the PR template's **🤖 AI Code Generation** section. There are three levels:

| Level | Description |
|---|---|
| 🤖 Fully AI-generated | AI wrote the code; contributor reviewed and validated it |
| 🛠️ Mostly AI-generated | AI produced the draft; contributor made significant modifications |
| 👨‍💻 Mostly Human-written | Contributor led; AI provided suggestions or none at all |

Honest disclosure is expected. There is no stigma attached to any level — what matters is the quality of the contribution.

### You Are Responsible for What You Submit

Using AI to generate code does not reduce your responsibility as the contributor. Before opening a PR with AI-generated code, you must:

- **Read and understand** every line of the generated code.
- **Test it** in a real environment (see the Test Environment section of the PR template).
- **Check for security issues** — AI models can generate subtly insecure code (e.g., path traversal, injection, credential exposure). Review carefully.
- **Verify correctness** — AI-generated logic can be plausible-sounding but wrong. Validate the behavior, not just the syntax.

PRs where it is clear the contributor has not read or tested the AI-generated code will be closed without review.

### AI-Generated Code Quality Standards

AI-generated contributions are held to the **same quality bar** as human-written code:

- It must pass all CI checks (`make check`).
- It must be idiomatic Go and consistent with the existing codebase style.
- It must not introduce unnecessary abstractions, dead code, or over-engineering.
- It must include or update tests where appropriate.

### Security Review

AI-generated code requires extra security scrutiny. Pay special attention to:

- File path handling and sandbox escapes (see commit `244eb0b` for a real example)
- External input validation in channel handlers and tool implementations
- Credential or secret handling
- Command execution (`exec.Command`, shell invocations)

If you are unsure whether a piece of AI-generated code is safe, say so in the PR — reviewers will help.

---
