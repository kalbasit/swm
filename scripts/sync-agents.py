#!/usr/bin/env python3
import os
import re
from pathlib import Path

# --- Configuration ---
BASE_DIR = Path(__file__).parent.parent
CLAUDE_DIR = BASE_DIR / ".claude"
GITHUB_DIR = BASE_DIR / ".github"
AGENT_DIR = BASE_DIR / ".agent"
TARGET_FILE = BASE_DIR / "AGENTS.md"

OPENSPEC_PREFIX = "openspec-"

# Constructed programmatically so Markdown/UI clipboards don't strip them as HTML tags
GITNEXUS_START = "<" + "!-- gitnexus:start --" + ">"
GITNEXUS_END = "<" + "!-- gitnexus:end --" + ">"
RULES_START = "<" + "!-- project-rules:start --" + ">"
RULES_END = "<" + "!-- project-rules:end --" + ">"


def get_gitnexus_block(text):
    """Extracts the GitNexus block if it exists."""
    pattern = re.escape(GITNEXUS_START) + r".*?" + re.escape(GITNEXUS_END)
    match = re.search(pattern, text, re.DOTALL)
    return match.group(0) if match else ""


def generate_rules_block():
    """Reads .claude/rules/*.md and wraps them in markers."""
    rules_path = CLAUDE_DIR / "rules"
    if not rules_path.exists():
        return ""

    block = [
        RULES_START,
        "# Project Rules",
        "These rules are authoritative for all agents.\n",
    ]

    for rule_file in sorted(rules_path.glob("*.md")):
        # Guard against hidden files and self-reference
        if rule_file.name.startswith(".") or rule_file.name in [
            "AGENTS.md",
            "CLAUDE.md",
        ]:
            continue

        title = rule_file.stem.replace("-", " ").title()
        content = rule_file.read_text().strip()
        block.append(f"## {title}\n{content}\n")

    block.append(RULES_END)
    return "\n".join(block)


def rebuild_agents():
    """
    Reconstructs the file from scratch by only grabbing known-good blocks.
    This guarantees zero bloat retention and preserves GitNexus context.
    """
    print(f"🧹 Rebuilding {TARGET_FILE.name}...")

    original_text = TARGET_FILE.read_text() if TARGET_FILE.exists() else ""

    # 1. Grab GitNexus (if it exists)
    gitnexus_part = get_gitnexus_block(original_text)

    # 2. Grab Rules
    rules_part = generate_rules_block()

    final_content = []

    # Put GitNexus first
    if gitnexus_part:
        final_content.append(gitnexus_part)
        print("  ✅ Preserved GitNexus block.")
    else:
        print("  ⚠️ No GitNexus block found to preserve.")

    # Put Rules second
    if rules_part:
        final_content.append(rules_part)
        print("  ✅ Generated Rules block.")

    # Write directly to the canonical file
    TARGET_FILE.write_text("\n\n".join(final_content) + "\n")
    print(f"✨ {TARGET_FILE.name} successfully updated.")


def sync_skills():
    """Symlinks non-OpenSpec skills."""
    print("🔄 Aligning skill symlinks...")
    skills_src = CLAUDE_DIR / "skills"
    if not skills_src.exists():
        return

    targets = [GITHUB_DIR / "skills", AGENT_DIR / "skills"]
    for skill_folder in skills_src.iterdir():
        if not skill_folder.is_dir() or skill_folder.name.startswith(OPENSPEC_PREFIX):
            continue

        for t_dir in targets:
            t_dir.mkdir(parents=True, exist_ok=True)
            target_link = t_dir / skill_folder.name

            # If it's a real directory (likely OpenSpec), leave it alone
            if target_link.exists() and not target_link.is_symlink():
                continue

            rel_src = os.path.relpath(skill_folder, t_dir)
            if target_link.is_symlink():
                target_link.unlink()

            target_link.symlink_to(rel_src, target_is_directory=True)
            print(
                f"  🔗 Linked {skill_folder.name} -> {target_link.relative_to(BASE_DIR)}"
            )


if __name__ == "__main__":
    try:
        rebuild_agents()
        sync_skills()
        print("\n✅ Sync complete. All systems aligned.")
    except Exception as e:
        print(f"❌ Error: {e}")
        exit(1)
