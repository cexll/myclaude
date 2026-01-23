# feature-dev

7 阶段功能开发工作流，使用 codeagent-wrapper 编排多个 agent。

## 安装

```bash
python install.py --module feature-dev
```

安装内容：
- `~/.claude/skills/feature-dev/` - skill 文件
- hooks 自动合并到 `~/.claude/settings.json`

## 使用

```
/feature-dev <功能描述>
```

示例：
```
/feature-dev 添加用户登录功能
/feature-dev 实现订单导出 CSV
```

## 工作流阶段

| 阶段 | 名称 | 目标 |
|------|------|------|
| 1 | Discovery | 理解需求 |
| 2 | Exploration | 探索代码库 |
| 3 | Clarification | 澄清疑问（必须） |
| 4 | Architecture | 设计方案 |
| 5 | Implementation | 实现（需审批） |
| 6 | Review | 代码审查 |
| 7 | Summary | 总结文档 |

## Agents

- `code-explorer` - 代码追踪、架构映射
- `code-architect` - 方案设计、文件规划
- `code-reviewer` - 代码审查、简化建议
- `develop` - 实现代码、运行测试

Agent 提示词位于 `agents/` 目录。如需自定义，可在 `~/.codeagent/agents/` 创建同名文件覆盖。

## ~/.codeagent/models.json 配置

可选。默认使用 codeagent-wrapper 内置配置。如需自定义 agent 模型：

```json
{
  "agents": {
    "code-explorer": {
      "backend": "claude",
      "model": "claude-sonnet-4-5-20250929"
    },
    "code-architect": {
      "backend": "claude",
      "model": "claude-sonnet-4-5-20250929"
    },
    "code-reviewer": {
      "backend": "claude",
      "model": "claude-sonnet-4-5-20250929"
    }
  }
}
```

## Loop 机制

安装后会注册 Stop hook。当 `/feature-dev` 执行时：

1. 创建 `.claude/feature-dev.local.md` 状态文件
2. 每阶段更新 `current_phase`
3. Stop hook 检测状态，未完成时阻止退出
4. 完成后输出 `<promise>FEATURE_COMPLETE</promise>` 结束

手动退出：将状态文件中 `active` 设为 `false`。

## 卸载

```bash
python install.py --uninstall --module feature-dev
```
