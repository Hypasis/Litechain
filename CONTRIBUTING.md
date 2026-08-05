# 🤝 Contributing to Litechain L1

Thank you for your interest in contributing to **Litechain L1**! We welcome contributions from developers, researchers, and community members.

To maintain code quality, security standards, and protocol integrity, **direct pushes to the `main` branch are strictly prohibited**. All contributions must be submitted via Pull Requests (PRs) from a forked repository and approved by the repository owner (`sanketsaagar`).

---

## 📋 **Contribution Workflow**

Follow these steps to contribute to Litechain:

### **1. Fork the Repository**
Navigate to [https://github.com/sanketsaagar/Litechain](https://github.com/sanketsaagar/Litechain) and click **Fork** in the top-right corner to create your own copy of the repository under your GitHub account.

### **2. Clone Your Fork**
```bash
git clone https://github.com/YOUR_GITHUB_USERNAME/Litechain.git
cd Litechain
git remote add upstream https://github.com/sanketsaagar/Litechain.git
```

### **3. Create a Feature Branch**
Create a new descriptive branch for your changes (do not work directly on `main`):
```bash
git checkout -b feat/your-feature-name
# or for bug fixes:
git checkout -b fix/issue-description
```

### **4. Implement Changes & Add Tests**
- Ensure your code adheres to standard Go formatting (`gofmt`).
- Write comprehensive unit tests for any new features or bug fixes.
- Ensure zero breaking changes to existing EVM or ZK interfaces.

### **5. Run Local Verification**
Before committing, verify that your changes compile and all test suites pass cleanly:
```bash
# Build all packages
go build ./...

# Run full test suite
go test -v ./pkg/...
```

### **6. Commit and Push to Your Fork**
```bash
git add .
git commit -m "feat(module): concise description of your changes"
git push origin feat/your-feature-name
```

### **7. Open a Pull Request (PR)**
1. Go to [https://github.com/sanketsaagar/Litechain](https://github.com/sanketsaagar/Litechain).
2. Click **New Pull Request** -> **Compare across forks**.
3. Select your fork and branch (`feat/your-feature-name`) as the head repository, and `sanketsaagar/Litechain` `main` as the base repository.
4. Fill out the PR template detailing:
   - What changes were made and why.
   - Test results and verification output.

---

## 🛡️ **PR Review & Approval Process**

1. **Owner Review**: Every PR is reviewed by the repository owner (`sanketsaagar`).
2. **Automated CI/CD Verification**: Automated build checks and test suites (`go test ./...`) must pass with zero warnings or errors.
3. **No Direct Pushes**: Direct commits to `main` are disabled via GitHub branch protection rules.
4. **Merge**: Once approved by the owner, the PR will be squashed and merged into `main`.

---

## ⚖️ **Code of Conduct & License**

By contributing to Litechain, you agree that your contributions will be licensed under the project's [MIT License](https://github.com/sanketsaagar/Litechain/blob/main/LICENSE).
