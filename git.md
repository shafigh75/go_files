This guide is designed to take you from a standard user to a "Senior Git User" capability, covering essential configurations, daily commands, disaster recovery, and DevOps-specific workflows.

## 1. Configuration & Setup
Before writing code, setting the environment correctly prevents attribution errors and improves efficiency.

| Command | One-Liner Description | Real World Example |
| :--- | :--- | :--- |
| `git config` | Sets configuration values at global, system, or local levels. | `git config --global user.name "John Doe"` (Sets identity for all repos). |
| `git config alias` | Creates shortcuts for long commands. | `git config --global alias.co checkout` (Now you can type `git co`). |
| `git config core.editor`| Sets the default text editor for commit messages. | `git config --global core.editor "code --wait"` (Opens VS Code). |

## 2. Starting & Staging (The Basics)

### `git init`
**Definition:** Initializes a new Git repository and creates the `.git` directory.
```bash
# Start tracking the current directory
git init
```

### `git clone`
**Definition:** Copies an existing repository from a remote source to your local machine.
```bash
# Clone a specific branch directly (saves time/bandwidth)
git clone -b develop https://github.com/user/repo.git
```

### `git add`
**Definition:** Moves changes from the working directory to the staging area (index).
```bash
# Add parts of a file interactively (patch mode - highly recommended for clean commits)
git add -p
```

### `git commit`
**Definition:** Captures a snapshot of the project's currently staged changes.
```bash
# Commit with a message and sign it with GPG (security best practice)
git commit -S -m "feat: add user login logic"
```

## 3. Inspection & Comparison

### `git status`
**Definition:** Displays the state of the working directory and the staging area.
```bash
# Show status in short format (less noise)
git status -s
```

### `git diff`
**Definition:** Shows changes between commits, commit and working tree, etc.
```bash
# Show changes in staged files only (what will be committed)
git diff --staged
```

### `git blame`
**Definition:** Shows what revision and author last modified each line of a file.
```bash
# Inspect a file, ignoring whitespace changes
git blame -w src/main.py
```

### `git log`
**Definition:** Shows the commit logs.
```bash
# A beautiful, graph-based visual of the history
git log --graph --oneline --all --decorate
```

## 4. Branching & Navigation

### `git checkout`
**Definition:** Switches branches or restores working tree files (older, overloaded command).
```bash
# Create a new branch and switch to it immediately
git checkout -b feature/new-api
```

### `git switch` (Modern alternative to checkout)
**Definition:** Specifically designed to switch branches.
```bash
# Switch to an existing branch
git switch develop
```

### `git restore` (Modern alternative to checkout)
**Definition:** Restore working tree files (discard changes).
```bash
# Unstage a file but keep changes in working directory
git restore --staged index.html
```

## 5. Syncing & Networking

### `git remote`
**Definition:** Manages the set of tracked repositories.
```bash
# Add a new remote (e.g., for a fork workflow)
git remote add upstream https://github.com/original/repo.git
```

### `git fetch`
**Definition:** Downloads objects and refs from another repository but **does not** integrate them into your working files.
```bash
# Fetch all branches from all remotes and prune deleted branches
git fetch --all --prune
```

### `git pull`
**Definition:** Fetches from and integrates with another repository or a local branch.
```bash
# Pull changes and rebase your local commits on top (keeps history linear)
git pull --rebase origin main
```

### `git push`
**Definition:** Updates remote refs along with associated objects.
```bash
# Push a new branch and set the upstream tracking information
git push -u origin feature/login
```

## 6. Advanced Integration (Merging & Rebasing)

### `git merge`
**Definition:** Joins two or more development histories together.
```bash
# Merge 'feature' into 'main' but force a merge commit (preserves feature existence in history)
git merge --no-ff feature/login
```

### `git rebase`
**Definition:** Reapplies commits on top of another base tip (rewrites history for a linear graph).
```bash
# Interactive rebase of last 3 commits (squash, edit, reword)
git rebase -i HEAD~3
```

### `git cherry-pick`
**Definition:** Applies the changes introduced by some existing commits.
```bash
# Apply a specific hotfix commit from 'main' into your 'release' branch
git cherry-pick a1b2c3d
```

## 7. Undoing & Cleaning (Safety Nets)

### `git reset`
**Definition:** Resets current HEAD to the specified state.
```bash
# Soft reset: Undo the commit but keep changes in staging area (fix a bad commit)
git reset --soft HEAD~1
```

### `git revert`
**Definition:** Creates a new commit that undoes the changes of a previous commit (safe for public history).
```bash
# Undo a specific commit without rewriting history
git revert a1b2c3d
```

### `git clean`
**Definition:** Removes untracked files from the working tree.
```bash
# Interactively remove untracked files and directories
git clean -idx
```

## 8. Context Switching & Sub-projects

### `git stash`
**Definition:** Temporarily shelves (stashes) changes so you can work on something else.
```bash
# Stash untracked files as well, with a message
git stash save -u "WIP: API refactoring"
```

### `git submodule`
**Definition:** Allows you to keep another Git repository in a specific subdirectory of your repository.
```bash
# Update submodules to the commit specified by the main repo
git submodule update --init --recursive
```

### `git worktree` (DevOps Superpower)
**Definition:** Allows managing multiple working trees attached to the same repository.
```bash
# create a new folder 'hotfix-folder' linked to the branch 'hotfix'
git worktree add ../hotfix-folder hotfix
```

## 9. Disaster Recovery

### `git reflog`
**Definition:** Tracks the tip of branches and other references that were updated in the local repository. *This allows you to recover "lost" commits.*
```bash
# Find the SHA of a commit you deleted with 'reset --hard' and restore it
git reflog
# then: git reset --hard <SHA-FROM-REFLOG>
```

---

# Nuances & Key Differences (The Senior Interview Section)

### 1. `git fetch` vs `git pull`
*   **Fetch:** goes to the internet, grabs the new data, and stores it in your local `.git` folder hidden away (`origin/main`). **It touches nothing in your code files.** Safe to run anytime.
*   **Pull:** runs `git fetch` first, and then immediately runs `git merge`. **It updates your code files.** If there are conflicts, `pull` breaks your flow.
*   **Senior Tip:** Prefer `fetch` then `diff` or `merge` manually in complex CI/CD environments to avoid surprise conflicts.

### 2. `git merge` vs `git rebase`
*   **Merge:** Creates a new "merge commit" tying two histories together. Preserves the exact history of events.
    *   *Pro:* Non-destructive, honest history.
    *   *Con:* History graph can look like "guitar hero" (messy).
*   **Rebase:** Picks up your commits and places them *after* the incoming changes.
    *   *Pro:* Perfectly linear history. Easier to debug with `git bisect`.
    *   *Con:* Rewrites history. **NEVER** rebase a public branch (like `main`) that others are working on.

### 3. `git reset` vs `git revert`
*   **Reset:** Moves the time pointer back. It's like the commit never happened.
    *   *Use case:* Local mistakes you haven't pushed yet.
*   **Revert:** Adds a *new* commit that does the exact opposite of the target commit.
    *   *Use case:* You pushed a bug to production and need to fix it immediately without breaking the history for other developers.

---

# Vital DevOps Workflows

### 1. The "Oh No, I Committed to Master" Fix
You accidentally committed to `main` (or `master`) locally, but you intended to create a new feature branch.
```bash
# 1. Create the new branch pointing to your current spot (with the accidental commits)
git branch feature/new-stuff

# 2. Reset 'main' back to where it should be (e.g., origin/main)
git reset --hard origin/main

# 3. Switch to your new branch
git switch feature/new-stuff
```

### 2. Debugging with `git bisect`
A bug appeared in production. You know it worked in version 1.0, but version 2.0 is broken. There are 100 commits in between.
```bash
# Start the binary search
git bisect start

# Tell git the current version is bad
git bisect bad

# Tell git a specific older commit was good
git bisect good <commit-hash-v1.0>

# Git will now jump to the middle. You test your app.
# If it works: git bisect good
# If it fails: git bisect bad
# Repeat until Git tells you exactly which commit introduced the bug.
```

### 3. Cleaning up Local Refs
Over time, your local machine thinks remote branches exist that have actually been deleted on GitHub/GitLab.
```bash
# Fetch and delete references to remote branches that no longer exist
git fetch -p 
# OR
git remote prune origin
```

### 4. Handling Large Repositories (Partial Clone)
If a repo is massive (GBs), a DevOps engineer shouldn't clone the whole history for a simple CI job.
```bash
# Clone only the latest commit (depth 1)
git clone --depth 1 https://github.com/org/massive-repo.git
```

### 5. Squashing Commits (Interactive Rebase)
Before merging a feature branch to main, clean up "wip", "typo", "fix" commits into one clean commit.
```bash
# Rebase the last X commits interactively
git rebase -i HEAD~5
# Change 'pick' to 'squash' (or 's') for the commits you want to merge into the top one.
```


Here is the advanced continuation of the guide. This section moves beyond standard daily usage into **automation, repo architecture, and "under the hood" mechanics**. Understanding these makes you the person the team calls when the repository is "broken."

## 10. Automation & Quality Control (Git Hooks)
Git hooks are scripts that run automatically every time a specific event occurs in a Git repository. For a DevOps engineer, these are vital for enforcing standards locally before code hits the CI pipeline.

| Command/Concept | One-Liner Description | Real World Example |
| :--- | :--- | :--- |
| `pre-commit` | Runs before a commit is created. Used to inspect the snapshot that's about to be committed. | Running a linter (e.g., `eslint` or `flake8`) to prevent bad code from entering history. |
| `commit-msg` | Runs after the commit message is entered but before the commit is finalized. | Checking that the message follows semantic versioning (e.g., must start with `feat:`, `fix:`, etc.). |
| `pre-push` | Runs during `git push`, after remote refs have been updated but before any objects are transferred. | Running unit tests to ensure you aren't pushing broken code to the remote. |

**Example: A Simple `pre-commit` Hook**
Create a file at `.git/hooks/pre-commit` and make it executable (`chmod +x`):
```bash
#!/bin/sh
# Block commits if they contain "DO NOT COMMIT" (useful for debugging code)
if git diff --cached | grep -q "DO NOT COMMIT"; then
    echo "Error: You have left 'DO NOT COMMIT' in your changes."
    exit 1
fi
```

## 11. Managing Large Files (Git LFS)
Standard Git is terrible at handling large binaries (images, videos, compiled binaries) because every version of that file is stored in history, bloating the clone size.

### `git lfs` (Large File Storage)
**Definition:** Replaces large files with text pointers inside Git, while storing the file contents on a remote server.
```bash
# 1. Install LFS in the repo
git lfs install

# 2. Track large files (e.g., Photoshop files)
git lfs track "*.psd"

# 3. Add the attributes file created by LFS
git add .gitattributes
```
**DevOps Note:** If your CI/CD pipeline is slow because cloning takes 10 minutes, checking for binary files committed without LFS is usually the first step to fixing it.

## 12. Repository Architecture & Dependency Management

### `git subtree`
**Definition:** An alternative to submodules that merges the sub-project's code directly into your main project's tree.
**Why use it?** Unlike submodules, users cloning your repo don't need to learn extra commands. The code is just *there*.
```bash
# Add a remote project into a folder named 'plugins/auth'
git subtree add --prefix plugins/auth https://github.com/other/auth-lib.git main --squash
```

### `.gitattributes`
**Definition:** A text file that gives attributes to pathnames.
**Why use it?** To enforce consistency across different operating systems (Windows vs. Linux/Mac).
```bash
# Force all text files to use LF (Linux style) line endings, even on Windows
* text=auto eol=lf

# Tell GitHub to treat .rb files as Ruby code for language statistics
*.rb linguist-language=Ruby

# Tell git diff to ignore generated minified files
jquery.min.js -diff
```

## 13. "Hidden" Gems for Power Users

### `git rerere` (Reuse Recorded Resolution)
**Definition:** Remembers how you resolved a hunk conflict so that the next time it sees the same conflict, it can resolve it automatically.
**Scenario:** You are constantly rebasing a long-lived feature branch against `main`. You get the same conflict every day. `rerere` fixes it for you after you fix it once.
```bash
# Enable it globally
git config --global rerere.enabled true
```

### `git shortlog`
**Definition:** Summarizes `git log` output, grouping commits by author.
**Scenario:** Generating release notes or seeing who contributed the most to the release.
```bash
# Show number of commits per author, sorted by number
git shortlog -sn --no-merges
```

### `git archive`
**Definition:** Creates a zip or tar archive of the files in a specific reference (branch/tag).
**Scenario:** Exporting the source code for a deployment artifact without the `.git` folder history.
```bash
# Zip the 'main' branch
git archive --format=zip --output=release.zip main
```

## 14. Git Internals (The "Plumbing")
To truly be a senior, you must understand that Git is essentially a content-addressable filesystem.

*   **Blob:** The content of a file. (If two files have the exact same content, they share the same Blob).
*   **Tree:** Represents a directory. Stores filenames and pointers to Blobs or other Trees.
*   **Commit:** Points to a Tree (the snapshot), the parent Commit(s), author, and message.
*   **Ref (Reference):** A sticky note (like "main" or "HEAD") that points to a specific Commit hash.

**DevOps Debugging Trick:**
If a file is corrupted or you want to see exactly what Git stored:
```bash
# Pretty-print the content of a git object (SHA)
git cat-file -p <object-hash>
```

## 15. Maintenance & Health

### `git gc` (Garbage Collection)
**Definition:** Cleans up unnecessary files and optimizes the local repository.
**Real World:** Git does this automatically occasionally, but if commands feel slow, run this manually.
```bash
# Aggressive clean (takes longer, packs better)
git gc --aggressive
```

### `git fsck` (File System Check)
**Definition:** Verifies the connectivity and validity of the objects in the database.
**Real World:** Use this if you suspect disk corruption or if a repo is acting weirdly.
```bash
# Check for full integrity and show dangling objects (deleted data that is still recoverable)
git fsck --full --no-reflogs
```

---

# DevOps Specific Git Strategy flows

### 1. The "Git Flow" vs. "Trunk Based Development"
A Senior Git user doesn't just know commands; they know **strategy**.

*   **Git Flow:** (Old School) `master`, `develop`, `feature/*`, `release/*`, `hotfix/*` branches.
    *   *Good for:* Scheduled release cycles (software sold on CDs/App Stores).
    *   *Bad for:* CI/CD and rapid web deployment (too much merging ceremony).
*   **Trunk Based Development:** (DevOps Standard) Developers commit to `main` (trunk) frequently (at least daily). Feature flags hide unfinished work.
    *   *Good for:* High-performing DevOps teams, CI/CD, rapid iteration.

### 2. Handling Secrets in Git (The Disaster Scenario)
Someone committed an AWS Key or Database Password to the repo. **`git rm` is not enough** because the key still lives in the history.

**The Fix (BFG Repo-Cleaner or git-filter-repo):**
*Note: Do not use `git filter-branch` (it is deprecated and slow).*

```bash
# Using git-filter-repo (Python tool, highly recommended)
# Remove the passwords.txt file from ALL history
git filter-repo --path passwords.txt --invert-paths
```
*After this, you must Force Push. This changes all commit hashes. You must coordinate with the whole team to re-clone the repo.*

### 3. The "Empty Commit" Trigger
Sometimes you need to trigger a CI/CD pipeline (e.g., to re-run a deployment) but you don't have code changes to make.
```bash
# Create a commit with no changes
git commit --allow-empty -m "trigger: re-run ci pipeline"
git push
```

### 4. Semantic Versioning Tagging
DevOps pipelines often rely on tags to determine release versions.
```bash
# Light tag (just a pointer)
git tag v1.0.0

# Annotated tag (stores full object with author/date - Recommended for Releases)
git tag -a v1.0.1 -m "Hotfix for production login bug"

# Push tags to remote (tags do not push automatically!)
git push --tags
```

## Checklist: When can you count yourself as a Senior Git User?

1.  **Safety:** You are no longer afraid of `rebase` or detached HEAD states because you know how `reflog` works.
2.  **History:** You know how to keep a history linear and clean using `squash` and interactive rebase.
3.  **Internals:** You understand that branches are just pointers to commits, not physical folders.
4.  **Triage:** You can use `bisect` to find a bug introduced 500 commits ago in minutes.
5.  **Hooks:** You implement local hooks to save the CI/CD server from wasting time on syntax errors.
6.  **Discipline:** You never force push (`push -f`) to a shared branch unless explicitly coordinating with the team, and you prefer `push --force-with-lease` (which checks if anyone else pushed while you were working).
