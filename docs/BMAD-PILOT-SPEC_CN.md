#!/usr/bin/env markdown
# `/bmad-pilot` 工作流与代理编排规范（中文）

> 面向企业的角色化敏捷工作流，包含审批门和质量门控。本文汇总 `/bmad-pilot` 命令的端到端逻辑，并映射各阶段所对应的 Agent 职责与产物。

## 1. 概览

- 目标：用专业角色代理自动化完整的敏捷闭环：产品负责人（PO）、架构师、Scrum Master（SM）、开发（Dev）、独立评审（Review）、QA。
- 原则：关键设计点设置质量门（≥90 分）与用户审批门；仓库上下文驱动；迭代澄清。
- 产物路径：`./.claude/specs/{feature_name}/`。
- 可选参数：
  - `--skip-tests`：跳过 QA 阶段
  - `--direct-dev`：跳过 SM 规划，架构后直接进入开发
  - `--skip-scan`：跳过仓库扫描（不推荐）

## 2. 命令

```
/bmad-pilot <PROJECT_DESCRIPTION> [--skip-tests] [--direct-dev] [--skip-scan]
```

输入处理
- 从 `<PROJECT_DESCRIPTION>` 生成 `feature_name`（kebab-case）。
- 确保 `./.claude/specs/{feature_name}/` 目录存在。
- 若输入超过 500 字：先生成摘要并向用户确认。
- 若输入不清晰：发起针对性澄清提问。

## 3. 阶段与审批门

### Phase 0 — 仓库扫描（默认开启，可用 `--skip-scan` 跳过）
- 目标：收集技术栈、目录结构、约定、依赖、CI/测试等上下文。
- 产物：`00-repo-scan.md`。

### Phase 1 — 产品需求（PO）
- 通过质量评分循环，直到 PRD ≥ 90。
- 产物（经用户确认后保存）：`01-product-requirements.md`。
- 审批门 #1：用户必须确认保存并同意进入下一阶段。

### Phase 2 — 系统架构（Architect）
- 通过质量评分循环，直到架构 ≥ 90。
- 产物（经用户确认后保存）：`02-system-architecture.md`。
- 审批门 #2：用户必须确认保存并同意进入下一阶段。

### Phase 3 — Sprint 规划（SM）【`--direct-dev` 时跳过】
- 互动式规划：故事、任务、预估、风险；迭代至可执行。
- 产物（经用户确认后保存）：`03-sprint-plan.md`。
- 审批门 #3：用户必须确认计划后方可开始开发。

### Phase 4 — 开发实现（Dev）
- 按 PRD/架构/(Sprint 计划) 实现。
- 产物：代码变更写入仓库。

### Phase 4.5 — 代码评审（独立评审）
- 产物：`04-dev-reviewed.md`，状态：Pass / Pass with Risk / Fail。
- 循环：Fail 则回到 Dev 修复后再评审。

### Phase 5 — 质量保障（QA）【`--skip-tests` 时跳过】
- 基于 PRD 与架构创建并执行测试集。
- 产物：测试执行结果（可在报告或 CI 输出中体现）。

## 4. Agent 角色与职责

所有 Agent 都会读取先前产物与 `00-repo-scan.md` 仓库上下文。

- `bmad-po`（产品负责人）
  - 输入：项目描述、仓库上下文。
  - 输出：PRD（≥90）→ 用户批准后保存为 `01-product-requirements.md`。
  - 交互：提出澄清问题；由编排器中转问答与循环。

- `bmad-architect`（系统架构师）
  - 输入：PRD、仓库上下文。
  - 输出：架构设计（≥90）→ 用户批准后保存为 `02-system-architecture.md`。
  - 交互：就关键技术决策提问；由编排器中转。

- `bmad-sm`（Scrum Master）
  - 输入：PRD、架构设计、仓库上下文。
  - 输出：Sprint 计划 → `03-sprint-plan.md`（用户批准后保存）；`--direct-dev` 时跳过。

- `bmad-dev`（开发）
  - 输入：PRD、架构、（Sprint 计划）、仓库上下文。
  - 输出：工作代码与必要测试。

- `bmad-review`（独立评审）
  - 输入：实现代码 + 全部规格文档。
  - 输出：`04-dev-reviewed.md`（Pass/Risk/Fail）；Fail 则回流 Dev 修复。

- `bmad-qa`（QA 工程师）
  - 输入：实现代码 + 全部规格文档。
  - 输出：执行测试与结果；确保满足验收标准。

- `bmad-orchestrator`（编排器）
  - 输入：用户意图；负责协调各阶段、管理审批门、控制保存时机与产物一致性。

## 5. 质量与门控

- PRD 质量（≥90）→ 进入架构阶段。
- 架构质量（≥90）→ 进入 SM 或直接进入 Dev（`--direct-dev`）。
- Sprint 计划：用户审批后进入 Dev。
- 评审状态：
  - Pass → 进入 QA（除非 `--skip-tests`）。
  - Pass with Risk → 可选的后续建议。
  - Fail → 回到 Dev 修复并复审。

## 6. 产物清单

保存到 `./.claude/specs/{feature_name}/`：
```
00-repo-scan.md
01-product-requirements.md
02-system-architecture.md
03-sprint-plan.md            # 若未跳过
04-dev-reviewed.md
```

## 7. 执行逻辑（简化伪码）

```pseudo
parse_options()
feature = to_kebab_case(PROJECT_DESCRIPTION)
ensure_specs_dir(feature)

if not --skip-scan:
  scan_repo() -> write 00-repo-scan.md

prd_score = 0
while prd_score < 90:
  prd, prd_score = po_iterate()
user_approve_or_loop()
save('01-product-requirements.md', prd)

arch_score = 0
while arch_score < 90:
  arch, arch_score = architect_iterate()
user_approve_or_loop()
save('02-system-architecture.md', arch)

if not --direct-dev:
  sprint_plan = sm_iterate_until_actionable()
  user_approve_or_loop()
  save('03-sprint-plan.md', sprint_plan)

develop()
status = review()
while status == 'Fail':
  develop_fix()
  status = review()

if not --skip-tests:
  qa_execute()
finish()
```

## 8. 与 `/alin-dev` 的区别与选型建议

关键差异
- 范围与严谨度：BMAD 是完整敏捷流程，含 6 个角色代理与 3 个审批门（PRD、架构、Sprint 计划）。`/alin-dev` 以实现为中心，流程更轻量。
- 产物路径：BMAD 写入 `./.claude/specs/{feature_name}/`；`/alin-dev` 写入 `./.alin/specs/{feature_name}/`。
- 阶段：BMAD 在开发前包含 PO + Architect + SM；`/alin-dev` 以 `alin-generate` 规格 → `alin-code` → `alin-review` 为主，并默认新增 `alin-manual-validate`（可指导手工验证）与 `alin-testing`（按需）。
- 额外交付：`/alin-dev` 默认生成 `requirements-manual-valid.md`（手动验证指南），BMAD 不生成此文档。
- 选项差异：BMAD 有 `--direct-dev` 跳过 SM；`/alin-dev` 有 `--skip-manual` 跳过手动验证指南生成。

何时用哪一个
- 选择 `/bmad-pilot` 当：
  - 需求复杂、涉及多方或存在高不确定性。
  - 架构决策风险较高，需要明确设计评审与把关。
  - 需要 Sprint 规划与正式审批门，强调组织/团队对齐与过程文档化。
- 选择 `/alin-dev` 当：
  - 任务清晰、以实现速度为优先。
  - 希望用一份具体技术规格直接驱动编码。
  - 更偏好生成“手动验证指南”以端到端验收。
  - 功能规模中等，不需要单独的 Architect/SM 阶段。

快速判断
```
“复杂 + 不确定 + 高风险” → /bmad-pilot
“清晰 + 快速交付 + 实用验证” → /alin-dev
```

## 9. 建议与实践

- 不要随意跳过仓库扫描，除非你已充分了解项目上下文。
- 通过审批门确保质量与干系人对齐。
- 及时更新产物，确保每一阶段文档反映最新决策。
- 每个阶段都以小步迭代提高质量，再进行保存与推进。

---

参考
- 命令源文件：`bmad-agile-workflow/commands/bmad-pilot.md`
- 工作流指南：`docs/BMAD-WORKFLOW.md`
- alin-dev 规范：`alin-dev-workflow/commands/alin-dev.md`
