# Octo Flow — Bot + 人 可视化编排引擎

> 产品设计文档 v1.0 | 2026-05-26 | Author: Coda for Yu

---

## 一、定位与愿景

**一句话**：让 Octo 用户用拖拽或自然语言，把 Bot 和人编排成自动化工作流。

Octo 已经有了 Bot 体系（Coda、TestBot、OpsBot…）和 IM 群组。但目前 Bot 之间的协作、Bot 和人之间的流程串联，全靠硬编码（Go dispatcher）或手工协调。Octo Flow 把这层逻辑抽象成**可视化、可复用、可分享**的编排层。

**差异化**：
- n8n / Windmill / Kestra 编排的是 API 和函数 → **Octo Flow 编排的是 IM 里的 Bot 和人**
- 执行界面不是 dashboard → **执行界面就是聊天窗口**——人在群里收到任务、在群里回复，flow 自动继续
- 编排不只是开发者工具 → **自然语言建 flow + 模板市场**，运营、HR、客服都能用

---

## 二、整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                        Octo Flow 架构                        │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐  │
│  │  可视化编排器  │  │ 自然语言入口  │  │    模板市场       │  │
│  │  (Web Editor) │  │ (NL→Flow)    │  │  (Marketplace)   │  │
│  └──────┬───────┘  └──────┬───────┘  └────────┬─────────┘  │
│         │                 │                    │             │
│  ───────┴─────────────────┴────────────────────┴──────────  │
│                     Flow Definition API                      │
│  ────────────────────────────────────────────────────────── │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐   │
│  │              Flow Engine（执行引擎）                    │   │
│  │                                                      │   │
│  │  ┌─────────┐ ┌──────────┐ ┌────────┐ ┌───────────┐  │   │
│  │  │ Trigger  │ │ Router   │ │ Runner │ │ State Mgr │  │   │
│  │  │ Manager  │ │ (DAG)    │ │        │ │           │  │   │
│  │  └─────────┘ └──────────┘ └────────┘ └───────────┘  │   │
│  │                                                      │   │
│  │  ┌─────────┐ ┌──────────┐ ┌────────────────────┐    │   │
│  │  │ Timeout  │ │ Retry &  │ │ Execution Logger   │    │   │
│  │  │ Escalate │ │ Fallback │ │                    │    │   │
│  │  └─────────┘ └──────────┘ └────────────────────┘    │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                             │
│  ────────────────────────────────────────────────────────── │
│                    Participant Adapter Layer                  │
│  ────────────────────────────────────────────────────────── │
│                                                             │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌───────────┐  │
│  │ Bot      │  │ Member   │  │ Contact  │  │ Role      │  │
│  │ Adapter  │  │ Adapter  │  │ Adapter  │  │ Resolver  │  │
│  │          │  │          │  │          │  │           │  │
│  │ 调 Bot   │  │ 群内发消息│  │ 私聊发   │  │ 运行时解析│  │
│  │ API 执行 │  │ 等回复    │  │ 消息等回复│  │ 角色→具体人│  │
│  └──────────┘  └──────────┘  └──────────┘  └───────────┘  │
│                                                             │
│  ────────────────────────────────────────────────────────── │
│                    Integration Layer                         │
│  ────────────────────────────────────────────────────────── │
│                                                             │
│  ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐ ┌──────────┐ │
│  │ GitHub │ │ GitLab │ │ Jira   │ │ Feishu │ │ Custom   │ │
│  │ Events │ │ Events │ │ Events │ │ Events │ │ Webhook  │ │
│  └────────┘ └────────┘ └────────┘ └────────┘ └──────────┘ │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

四层分工：
1. **表现层**：可视化编辑器 + 自然语言 + 模板市场 — 面向用户
2. **引擎层**：DAG 执行、状态管理、超时、重试 — 完全通用，不含业务语义
3. **参与者适配层**：Bot / 人 / 角色的统一抽象 — Octo 特有能力
4. **集成层**：外部系统对接 — 可扩展

---

## 三、核心概念

### 3.1 Flow（工作流）

一个 Flow 是一个 DAG（有向无环图），由节点（Node）和连线（Edge）组成。

```yaml
# Flow 定义 schema
flow:
  id: "flow_pr_review"
  name: "PR Review 自动化"
  version: 3
  description: "GitHub PR → 自动审查 → 通知"
  space_id: "sp_xxx"           # 所属 Space
  created_by: "uid_xxx"
  
  # 触发器（一个 flow 可以有多个触发器）
  triggers:
    - type: webhook
      id: "trg_github_pr"
      config:
        path: "/flow/pr-review"
        secret: "${GITHUB_WEBHOOK_SECRET}"
        signature_header: "X-Hub-Signature-256"
        signature_algo: "hmac-sha256"
    - type: cron
      id: "trg_daily_check"
      config:
        expression: "0 9 * * 1-5"
        timezone: "Asia/Shanghai"
    - type: message
      id: "trg_keyword"
      config:
        match: "/review {pr_url}"
        channels: ["group:xxx"]
  
  # 变量（flow 级别）
  variables:
    github_token: "${secret:GITHUB_TOKEN}"
    review_timeout: "2h"
  
  # 节点定义
  nodes: [...]
  
  # 连线定义
  edges: [...]
```

### 3.2 Node（节点）

八种节点类型，覆盖所有场景：

```yaml
nodes:
  # 1. Bot 动作节点 — 调用 Octo Bot 执行任务
  - id: "n_codex_review"
    type: bot_action
    config:
      bot_id: "testbot_bot"
      action: "codex_review"        # Bot 声明的能力之一
      input:
        repo: "{{trigger.payload.repository.full_name}}"
        pr_number: "{{trigger.payload.number}}"
      timeout: "30m"
      retry: { max: 2, delay: "1m" }
  
  # 2. 人工节点 — 发消息给人，等回复
  - id: "n_approval"
    type: human_action
    config:
      participant:
        type: member              # member | contact | role
        id: "uid_yu"              # 具体 UID
        # 或: type: role, id: "tech_lead" → 运行时解析
      channel: "group:xxx"        # 在哪个群发消息
      message: "PR #{{vars.pr_number}} 已通过自动审查，是否部署到生产？\n回复「部署」确认，「拒绝」取消"
      expect:
        type: keyword             # keyword | button | reaction | any
        keywords:
          approve: ["部署", "确认", "approve", "yes"]
          reject: ["拒绝", "取消", "reject", "no"]
      timeout:
        duration: "2h"
        remind: "30m"             # 每 30 分钟提醒一次
        max_remind: 3
        escalate_to: "uid_leader" # 超时升级给谁
        default: "reject"         # 超时默认走哪条路
  
  # 3. 条件分支
  - id: "n_check_pr_type"
    type: condition
    config:
      expression: "{{n_classify.output.complexity}}"
      branches:
        - value: "trivial"
          label: "简单 PR"
        - value: "complex"
          label: "复杂 PR"
        - default: true
          label: "其他"
  
  # 4. 并行节点 — 同时执行多个分支
  - id: "n_parallel_review"
    type: parallel
    config:
      mode: all                   # all = 等所有完成 | any = 任一完成即继续
      branches:
        - nodes: ["n_codex_review"]
        - nodes: ["n_security_scan"]
  
  # 5. 循环 / 轮询节点
  - id: "n_wait_merge"
    type: loop
    config:
      condition: "{{n_check_status.output.merged}} != true"
      max_iterations: 60
      interval: "1m"
      body_nodes: ["n_check_status"]
  
  # 6. HTTP 请求节点
  - id: "n_post_status"
    type: http
    config:
      method: POST
      url: "https://api.github.com/repos/{{vars.repo}}/statuses/{{vars.sha}}"
      headers:
        Authorization: "Bearer {{vars.github_token}}"
      body:
        state: "pending"
        context: "octo-flow/review"
  
  # 7. 代码块节点 — 跑任意脚本
  - id: "n_classify"
    type: script
    config:
      runtime: javascript         # javascript | python | bash
      code: |
        const files = input.files;
        const loc = files.reduce((sum, f) => sum + f.changes, 0);
        if (loc < 20 && files.length <= 2) return { complexity: 'trivial' };
        if (loc > 500 || files.length > 20) return { complexity: 'complex' };
        return { complexity: 'normal' };
      input:
        files: "{{n_fetch_files.output.files}}"
  
  # 8. 子流程调用
  - id: "n_notify_all"
    type: subflow
    config:
      flow_id: "flow_notify_team"
      input:
        message: "{{n_summary.output.text}}"
```

### 3.3 Edge（连线）

```yaml
edges:
  - from: "n_parse"
    to: "n_filter"
  
  - from: "n_filter"
    to: "n_classify"
    condition: "{{n_filter.output.should_process}} == true"
  
  - from: "n_filter"
    to: "__end__"
    condition: "{{n_filter.output.should_process}} == false"
    label: "跳过"
  
  # 条件分支的出口
  - from: "n_check_pr_type"
    to: "n_quick_review"
    branch: "trivial"
  
  - from: "n_check_pr_type"
    to: "n_full_review"
    branch: "complex"
```

### 3.4 Participant（参与者）

统一的参与者类型系统：

```yaml
participant_types:
  # Bot — 自动执行，通过 Bot API 调用
  bot:
    id: "testbot_bot"
    resolve: static                # 编排时确定
    execute: api_call              # 走 Bot Action API
    capabilities: [...]            # 从 Bot Capability Registry 读取
  
  # 成员 — 群内的人
  member:
    id: "uid_xxx"
    resolve: static
    execute: im_message            # 发 IM 消息，等回复
    channel: "group:xxx"           # 指定在哪个群 @ 他
  
  # 联系人 — 不在群内，通过私聊
  contact:
    id: "uid_yyy"
    resolve: static
    execute: im_dm                 # 发私聊消息，等回复
  
  # 角色 — 运行时动态解析
  role:
    id: "tech_lead"
    resolve: dynamic               # 运行时从 Space 角色表解析
    fallback: "uid_default"        # 解析不到时的兜底
    execute: im_message
```

---

## 四、参与者体系深度设计

### 4.1 Bot Capability Registry

每个 Octo Bot 注册时声明自己能做什么，编排器和 LLM 都依赖这个注册表：

```yaml
# Bot 能力声明（Bot 开发者提供）
bot_capability:
  bot_id: "testbot_bot"
  name: "TestBot"
  description: "代码审查和静态分析 Bot"
  
  actions:
    - id: "codex_review"
      name: "Codex 代码审查"
      description: "对 PR 进行 AI 辅助代码审查，输出 findings"
      input_schema:
        type: object
        properties:
          repo: { type: string, description: "仓库全名 org/repo" }
          pr_number: { type: integer }
          focus: { type: string, enum: [security, performance, all] }
        required: [repo, pr_number]
      output_schema:
        type: object
        properties:
          verdict: { type: string, enum: [approve, request_changes, comment] }
          findings: { type: array, items: { type: object } }
          summary: { type: string }
      estimated_duration: "10-30m"
      
    - id: "browse_url"
      name: "浏览网页"
      description: "访问 URL 并提取内容"
      input_schema:
        type: object
        properties:
          url: { type: string }
        required: [url]
      output_schema:
        type: object
        properties:
          content: { type: string }
          title: { type: string }
```

### 4.2 人工节点交互流程

```
Flow 执行到 human_action 节点
        │
        ▼
┌─────────────────────────┐
│ 发送消息给指定参与者       │
│ (群里 @ 或私聊)           │
│                         │
│ "PR #42 已通过审查，      │
│  回复「部署」确认"         │
└────────────┬────────────┘
             │
     ┌───────┴───────┐
     │   等待回复     │ ← Flow 在此暂停，释放资源
     │               │    消息回调唤醒
     └───┬───────┬───┘
         │       │
    回复匹配   超时
         │       │
         ▼       ▼
   ┌─────────┐  ┌──────────────┐
   │ 继续执行 │  │ 30m: 第 1 次提醒 │
   │ 下一节点 │  │ 60m: 第 2 次提醒 │
   └─────────┘  │ 90m: 第 3 次提醒 │
                │ 120m: 升级给 leader│
                │  或走 default 路径 │
                └──────────────┘
```

**关键设计：消息回调唤醒**

人工节点不是轮询，而是：
1. Flow Engine 发完消息后，在 State Manager 里注册一个 **wait handle**：`{execution_id, node_id, expect_from: uid, expect_channel: group_id}`
2. Octo IM 消息管道里加一个 **Flow Interceptor** 中间件
3. 群里有新消息时，Interceptor 检查是否有 wait handle 匹配（发送人 + 频道）
4. 匹配到 → 提取消息内容 → 对照 expect 规则 → 匹配关键词/按钮 → 唤醒 flow 继续
5. 不匹配 → 正常消息流，不影响聊天

这样人在群里正常聊天，reply 被 flow 捕获并推进流程——**对话即界面**。

### 4.3 角色动态解析

```yaml
# Space 角色配置（管理后台维护）
space_roles:
  - id: "tech_lead"
    name: "技术负责人"
    members: ["uid_yu"]
    
  - id: "qa_owner"
    name: "质量负责人"
    members: ["uid_jerry", "uid_test"]
    assign_strategy: round_robin  # round_robin | random | least_busy | all
    
  - id: "on_call"
    name: "值班人"
    source: schedule              # 从排班表动态取
    schedule_id: "sched_xxx"
```

编排时用角色，运行时解析为具体的人。好处：
- 人员变动不需要改 flow
- 支持多人轮转（round robin / 最闲 / 全员）
- 对接排班系统

---

## 五、触发器设计

```yaml
trigger_types:
  # 1. Webhook — 外部系统推送
  webhook:
    config:
      path: string               # URL path（自动生成或自定义）
      secret: string             # 签名密钥
      signature_header: string   # 签名 header 名
      signature_algo: string     # hmac-sha256 等
      filter: expression         # 可选：只接受符合条件的 payload
    output: trigger.payload      # 整个 JSON body

  # 2. Cron — 定时
  cron:
    config:
      expression: string         # 标准 5 字段 cron
      timezone: string
    output: trigger.scheduled_at

  # 3. 消息触发 — 群里有人说了某句话
  message:
    config:
      channels: [string]         # 监听哪些群/DM
      match:                     # 匹配规则
        type: keyword | regex | command | intent
        pattern: string
        # intent 类型会过 LLM 判断语义
    output: trigger.message      # 消息内容、发送人等

  # 4. 事件触发 — Octo 内部事件
  event:
    config:
      event_type: string         # member_join | bot_added | message_reaction | ...
      filter: expression
    output: trigger.event

  # 5. 手动触发
  manual:
    config:
      input_schema: object       # 手动触发时需要填的参数
    output: trigger.input

  # 6. 子流程调用（被其他 flow 调用）
  subflow:
    config:
      input_schema: object
    output: trigger.input
```

**消息触发 + intent 匹配**特别值得展开：

```yaml
# 例：用户在群里说"帮我查下这个 PR 的状态"
triggers:
  - type: message
    config:
      channels: ["group:dev_team"]
      match:
        type: intent
        intents:
          - name: "check_pr_status"
            examples:
              - "帮我查下 PR 状态"
              - "这个 PR 怎么样了"
              - "review 到哪了"
            extract:
              pr_url: "URL pattern"
              pr_number: "number after # or PR"
```

LLM 判断用户意图是否匹配 + 提取参数 → 触发 flow。这让非技术用户可以用自然语言触发工作流。

---

## 六、自然语言编排

### 6.1 整体流程

```
用户输入自然语言描述
        │
        ▼
┌─────────────────────────────┐
│ 1. Context 组装              │
│    - Bot Capability Registry │
│    - 当前 Space 成员列表      │
│    - 当前 Space 角色列表      │
│    - 可用集成列表             │
│    - Flow Schema 定义        │
└────────────┬────────────────┘
             │
             ▼
┌─────────────────────────────┐
│ 2. LLM 翻译                 │
│    System prompt:            │
│    "你是 Octo Flow 编排助手"  │
│    + context                 │
│    + 用户描述                 │
│    → 生成 Flow YAML          │
└────────────┬────────────────┘
             │
             ▼
┌─────────────────────────────┐
│ 3. Schema 校验               │
│    - YAML 格式合法            │
│    - 节点类型合法             │
│    - Bot action 存在          │
│    - 参与者 ID 存在           │
│    - 连线无环                 │
└────────────┬────────────────┘
             │
        校验通过
             │
             ▼
┌─────────────────────────────┐
│ 4. 渲染到可视化编辑器         │
│    用户看到 DAG 拓扑图        │
│    可以微调后保存              │
└─────────────────────────────┘
```

### 6.2 示例对话

**用户说：**
> 每次 GitHub 有新 PR，先让 TestBot 审代码。如果 TestBot 觉得没问题就自动 approve，如果有问题就 @ 余嘉伟 确认是否仍然合并。

**LLM 生成的 flow：**

```yaml
flow:
  name: "PR 自动审查"
  triggers:
    - type: webhook
      config:
        path: "/flow/auto-pr-review"
        signature_header: "X-Hub-Signature-256"
        signature_algo: "hmac-sha256"
  nodes:
    - id: parse
      type: script
      config:
        runtime: javascript
        code: |
          return {
            repo: input.payload.repository.full_name,
            pr: input.payload.number,
            sha: input.payload.pull_request.head.sha
          };
    - id: review
      type: bot_action
      config:
        bot_id: "testbot_bot"
        action: "codex_review"
        input:
          repo: "{{parse.output.repo}}"
          pr_number: "{{parse.output.pr}}"
        timeout: "30m"
    - id: check_verdict
      type: condition
      config:
        expression: "{{review.output.verdict}}"
        branches:
          - value: "approve"
          - value: "request_changes"
    - id: auto_approve
      type: http
      config:
        method: POST
        url: "https://api.github.com/repos/{{parse.output.repo}}/pulls/{{parse.output.pr}}/reviews"
        body: { event: "APPROVE" }
    - id: ask_yu
      type: human_action
      config:
        participant: { type: member, id: "uid_yu" }
        channel: "group:dev_team"
        message: |
          PR #{{parse.output.pr}} 审查发现问题：
          {{review.output.summary}}
          
          回复「合并」强制合并，「拒绝」关闭
        expect:
          type: keyword
          keywords:
            merge: ["合并", "merge"]
            reject: ["拒绝", "reject"]
        timeout: { duration: "4h", default: "reject" }
  edges:
    - from: parse → review
    - from: review → check_verdict
    - from: check_verdict → auto_approve (branch: approve)
    - from: check_verdict → ask_yu (branch: request_changes)
```

### 6.3 模板辅助生成

当用户说"我要一个新人入职流程"时，不是从零生成，而是：

1. 先从模板市场语义搜索匹配模板
2. 找到最相似的模板作为基础
3. 用 LLM 根据用户描述做定制（替换具体的 bot、人、群、参数）
4. 渲染给用户确认

命中率更高、幻觉更少。

---

## 七、模板市场

### 7.1 模板结构

```yaml
template:
  id: "tpl_pr_review_v2"
  name: "GitHub PR 自动审查"
  description: "PR 提交后自动触发代码审查，通过则 approve，不通过 @ 负责人"
  category: "DevOps"
  tags: ["github", "code-review", "ci"]
  author: "octo-official"         # 官方 or 社区用户
  downloads: 1234
  rating: 4.8
  
  # 模板参数 — 安装时用户填
  parameters:
    - id: "review_bot"
      name: "审查 Bot"
      type: bot
      description: "选择执行代码审查的 Bot"
      required: true
    - id: "approver"
      name: "审批人"
      type: participant             # bot | member | role
      description: "审查不通过时通知谁"
      required: true
    - id: "notify_channel"
      name: "通知群"
      type: channel
      required: true
    - id: "review_timeout"
      name: "审查超时"
      type: duration
      default: "30m"
  
  # 模板 flow 定义（占位符引用 parameters）
  flow:
    nodes:
      - id: review
        type: bot_action
        config:
          bot_id: "{{param.review_bot}}"
          action: "codex_review"
          timeout: "{{param.review_timeout}}"
      - id: ask_approver
        type: human_action
        config:
          participant: "{{param.approver}}"
          channel: "{{param.notify_channel}}"
    ...
```

### 7.2 市场功能

| 功能 | 描述 |
|---|---|
| **浏览** | 按分类（DevOps / HR / 客服 / 运营 / 财务）、标签、热门、最新 |
| **搜索** | 关键词 + 语义搜索 |
| **预览** | 查看 flow 拓扑图 + 参数说明，不需要安装 |
| **安装** | 一键导入，填参数（选 bot、选人、选群）→ 生成 flow |
| **发布** | 用户把自己的 flow 导出为模板，自动脱敏（UID → 占位符），审核后上架 |
| **版本** | 模板有版本，已安装的 flow 可以升级到新版本 |
| **Fork** | 在别人模板基础上改，发布为新模板 |

### 7.3 内置模板（官方提供）

| 模板 | 场景 |
|---|---|
| PR Review 自动化 | GitHub PR → Bot 审查 → 人确认 → merge |
| 新人入职流程 | HR 系统 webhook → 建群 → 开账号 → 分配 mentor |
| 值班轮转 | cron → 查排班 → @ 值班人 → 提醒交接 |
| 日报收集 | cron → 群发提醒 → 收集回复 → 汇总报告 |
| 客户工单 | 消息触发 → 分类 → 分配 → 超时升级 |
| 审批流程 | 申请 → 逐级审批 → 通知结果 |
| 告警处理 | webhook → 告警分级 → 通知值班 → 等确认 → 升级 |
| 内容审核 | 消息触发 → AI Bot 审核 → 人工复核 → 处置 |

---

## 八、执行引擎深度设计

### 8.1 Execution 状态机

```
                    ┌──────────┐
           ┌───────→│ RUNNING  │←────────┐
           │        └────┬─────┘         │
           │             │               │
     trigger/resume      │          wait 唤醒
           │             │               │
    ┌──────┴───┐    ┌────▼─────┐    ┌────┴──────┐
    │ PENDING  │    │ NODE_RUN │    │ WAITING   │
    │ (排队)    │    │ (执行中)  │    │ (等人回复) │
    └──────────┘    └────┬─────┘    └───────────┘
                         │
              ┌──────────┼──────────┐
              │          │          │
        ┌─────▼────┐ ┌──▼─────┐ ┌─▼────────┐
        │ SUCCESS  │ │ FAILED │ │ CANCELLED │
        │ (完成)    │ │ (失败)  │ │ (取消)    │
        └──────────┘ └────────┘ └───────────┘
```

### 8.2 Cancel 语义（cancel-on-push 场景）

```yaml
# Flow 级别的 cancel 策略
flow:
  concurrency:
    scope: "{{trigger.payload.pull_request.number}}"  # 按 PR 号分组
    strategy: cancel_previous    # cancel_previous | queue | reject_new
```

同一个 PR 有新 push → 新 execution 启动 → 自动 cancel 同 scope 的旧 execution。这个语义直接覆盖了 dispatcher 的 cancel-on-push。

### 8.3 数据流

节点之间通过 `{{node_id.output.field}}` 引用数据。引擎维护一个 execution context：

```json
{
  "execution_id": "exec_xxx",
  "flow_id": "flow_pr_review",
  "trigger": {
    "type": "webhook",
    "payload": { ... }
  },
  "nodes": {
    "n_parse": {
      "status": "success",
      "output": { "repo": "org/repo", "pr": 42, "sha": "abc123" },
      "started_at": "...",
      "finished_at": "..."
    },
    "n_review": {
      "status": "running",
      "output": null
    }
  }
}
```

---

## 九、数据模型

### 9.1 核心表

```sql
-- Flow 定义
CREATE TABLE flows (
    id          VARCHAR(36) PRIMARY KEY,
    space_id    VARCHAR(36) NOT NULL,
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    definition  JSONB NOT NULL,          -- 完整 flow YAML 转 JSON
    version     INTEGER DEFAULT 1,
    status      VARCHAR(20) DEFAULT 'draft',  -- draft | active | archived
    created_by  VARCHAR(36) NOT NULL,
    created_at  TIMESTAMP DEFAULT NOW(),
    updated_at  TIMESTAMP DEFAULT NOW()
);

-- Flow 版本历史
CREATE TABLE flow_versions (
    id          VARCHAR(36) PRIMARY KEY,
    flow_id     VARCHAR(36) REFERENCES flows(id),
    version     INTEGER NOT NULL,
    definition  JSONB NOT NULL,
    changelog   TEXT,
    created_at  TIMESTAMP DEFAULT NOW()
);

-- 触发器注册（运行时查询用）
CREATE TABLE triggers (
    id          VARCHAR(36) PRIMARY KEY,
    flow_id     VARCHAR(36) REFERENCES flows(id),
    type        VARCHAR(20) NOT NULL,     -- webhook | cron | message | event
    config      JSONB NOT NULL,
    webhook_path VARCHAR(255),            -- webhook 类型的 URL path（索引）
    status      VARCHAR(20) DEFAULT 'active',
    UNIQUE(webhook_path)
);

-- 执行实例
CREATE TABLE executions (
    id          VARCHAR(36) PRIMARY KEY,
    flow_id     VARCHAR(36) REFERENCES flows(id),
    trigger_id  VARCHAR(36) REFERENCES triggers(id),
    status      VARCHAR(20) NOT NULL,     -- pending | running | waiting | success | failed | cancelled
    context     JSONB NOT NULL,           -- 执行上下文（trigger data + node outputs）
    scope_key   VARCHAR(255),             -- 并发控制 key（如 PR 号）
    started_at  TIMESTAMP,
    finished_at TIMESTAMP,
    error       TEXT
);

-- 节点执行记录
CREATE TABLE node_executions (
    id            VARCHAR(36) PRIMARY KEY,
    execution_id  VARCHAR(36) REFERENCES executions(id),
    node_id       VARCHAR(100) NOT NULL,
    node_type     VARCHAR(30) NOT NULL,
    status        VARCHAR(20) NOT NULL,
    input         JSONB,
    output        JSONB,
    error         TEXT,
    started_at    TIMESTAMP,
    finished_at   TIMESTAMP
);

-- 等待句柄（人工节点暂停时注册）
CREATE TABLE wait_handles (
    id            VARCHAR(36) PRIMARY KEY,
    execution_id  VARCHAR(36) REFERENCES executions(id),
    node_id       VARCHAR(100) NOT NULL,
    expect_from   VARCHAR(36),            -- 期望谁回复（UID）
    expect_channel VARCHAR(36),           -- 期望在哪个频道
    expect_config JSONB,                  -- 匹配规则
    timeout_at    TIMESTAMP,
    status        VARCHAR(20) DEFAULT 'waiting',
    created_at    TIMESTAMP DEFAULT NOW()
);

-- Bot 能力注册
CREATE TABLE bot_capabilities (
    bot_id      VARCHAR(36) NOT NULL,
    action_id   VARCHAR(100) NOT NULL,
    name        VARCHAR(255),
    description TEXT,
    input_schema  JSONB,
    output_schema JSONB,
    estimated_duration VARCHAR(20),
    PRIMARY KEY (bot_id, action_id)
);

-- 模板市场
CREATE TABLE templates (
    id          VARCHAR(36) PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    category    VARCHAR(50),
    tags        TEXT[],
    author_id   VARCHAR(36),
    author_type VARCHAR(20),              -- official | community
    definition  JSONB NOT NULL,           -- 模板 flow 定义（含占位符）
    parameters  JSONB,                    -- 安装时需要填的参数
    version     VARCHAR(20),
    downloads   INTEGER DEFAULT 0,
    rating      DECIMAL(2,1),
    status      VARCHAR(20) DEFAULT 'pending',  -- pending | published | rejected
    created_at  TIMESTAMP DEFAULT NOW()
);

-- Space 角色
CREATE TABLE space_roles (
    id          VARCHAR(36) PRIMARY KEY,
    space_id    VARCHAR(36) NOT NULL,
    name        VARCHAR(100) NOT NULL,
    members     TEXT[],                    -- UID 列表
    assign_strategy VARCHAR(20) DEFAULT 'round_robin',
    schedule_id VARCHAR(36)               -- 可选关联排班
);
```

---

## 十、API 设计

### 10.1 Flow CRUD

```
POST   /api/v1/flows                    # 创建 flow
GET    /api/v1/flows                    # 列出 space 下所有 flow
GET    /api/v1/flows/:id                # 获取 flow 详情
PUT    /api/v1/flows/:id                # 更新 flow 定义
DELETE /api/v1/flows/:id                # 删除 flow
POST   /api/v1/flows/:id/activate       # 激活（注册触发器，开始监听）
POST   /api/v1/flows/:id/deactivate     # 停用
GET    /api/v1/flows/:id/versions       # 版本历史
```

### 10.2 执行

```
POST   /api/v1/flows/:id/execute        # 手动触发
GET    /api/v1/flows/:id/executions     # 执行历史列表
GET    /api/v1/executions/:id           # 执行详情（含每个节点状态）
POST   /api/v1/executions/:id/cancel    # 取消执行
POST   /api/v1/executions/:id/retry     # 从失败节点重试
```

### 10.3 Webhook 入口

```
POST   /api/v1/webhook/:path            # 统一 webhook 入口，path 匹配 trigger
```

### 10.4 Wait Handle（消息回调）

```
# 内部 API，Flow Interceptor 调用
POST   /api/v1/internal/wait-handles/match
       body: { from_uid, channel_id, message_content }
       response: { matched: true, execution_id, node_id, result: "approve" }
```

### 10.5 自然语言

```
POST   /api/v1/flows/generate
       body: { description: "每次 GitHub 有新 PR..." }
       response: { flow_definition: {...}, preview_url: "..." }

POST   /api/v1/flows/:id/edit-nl
       body: { instruction: "把超时从 2 小时改成 4 小时" }
       response: { updated_definition: {...}, diff: [...] }
```

### 10.6 模板市场

```
GET    /api/v1/templates                 # 浏览模板
GET    /api/v1/templates/:id             # 模板详情
POST   /api/v1/templates/:id/install     # 安装模板到 space
POST   /api/v1/templates                 # 发布模板
GET    /api/v1/templates/search?q=...    # 搜索模板
```

### 10.7 Bot Capability

```
POST   /api/v1/bots/:id/capabilities     # 注册 bot 能力
GET    /api/v1/bots/capabilities          # 列出所有 bot 能力（编排器用）
```

---

## 十一、Review 流程完整示例

用 Octo Flow 表达现有 Go dispatcher 的全部逻辑：

```yaml
flow:
  id: "flow_pr_review_dispatcher"
  name: "PR Review Dispatcher"
  version: 1

  concurrency:
    scope: "{{trigger.payload.repository.full_name}}/{{trigger.payload.number}}"
    strategy: cancel_previous     # ← 替代 cancel-on-push

  triggers:
    - type: webhook
      id: "github_pr"
      config:
        path: "/flow/pr-review"
        secret: "${GITHUB_WEBHOOK_SECRET}"
        signature_header: "X-Hub-Signature-256"
        signature_algo: "hmac-sha256"

  nodes:
    # 1. 解析 Payload
    - id: parse
      type: script
      config:
        runtime: javascript
        code: |
          const p = input.payload;
          const event = input.headers['x-github-event'];
          return {
            event,
            action: p.action,
            repo: p.repository.full_name,
            pr: p.number || p.pull_request?.number,
            sha: p.pull_request?.head?.sha || p.review?.commit_id,
            sender: p.sender.login,
            is_bot: p.sender.type === 'Bot',
            is_draft: p.pull_request?.draft || false,
            review_state: p.review?.state
          };

    # 2. 过滤：bot / draft / 不关心的 action
    - id: filter
      type: condition
      config:
        expression: |
          !{{parse.output.is_bot}} 
          && !{{parse.output.is_draft}}
          && ({{parse.output.action}} in ['opened', 'synchronize', 'submitted'])
        branches:
          - value: true
            label: "需处理"
          - value: false
            label: "跳过"

    # 3. 路由：PR event vs Review event
    - id: route
      type: condition
      config:
        expression: "{{parse.output.event}}"
        branches:
          - value: "pull_request"
          - value: "pull_request_review"

    # === PR Event 分支 ===

    # 4. Fetch changed files
    - id: fetch_files
      type: http
      config:
        method: GET
        url: "https://api.github.com/repos/{{parse.output.repo}}/pulls/{{parse.output.pr}}/files"
        headers:
          Authorization: "Bearer ${GITHUB_TOKEN}"

    # 5. 分类 PR
    - id: classify
      type: script
      config:
        runtime: javascript
        code: |
          const files = input.files;
          const loc = files.reduce((s, f) => s + f.changes, 0);
          if (loc < 20 && files.length <= 2) return { level: 'trivial' };
          if (loc > 500 || files.length > 20) return { level: 'complex' };
          return { level: 'normal' };

    # 6. Post GitHub pending status
    - id: post_pending
      type: http
      config:
        method: POST
        url: "https://api.github.com/repos/{{parse.output.repo}}/statuses/{{parse.output.sha}}"
        headers:
          Authorization: "Bearer ${GITHUB_TOKEN}"
        body:
          state: "pending"
          context: "octo-flow/review"
          description: "Review in progress..."

    # 7. 派发 Multica（Bot 动作节点）
    - id: dispatch_review
      type: bot_action
      config:
        bot_id: "codexbot"
        action: "review_pr"
        input:
          repo: "{{parse.output.repo}}"
          pr_number: "{{parse.output.pr}}"
          complexity: "{{classify.output.level}}"
        timeout: "45m"

    # 8. 通知 CR 子区
    - id: notify_cr
      type: bot_action
      config:
        bot_id: "echo_bot"
        action: "send_message"
        input:
          channel: "group:cr_thread"
          message: "🔍 PR #{{parse.output.pr}} ({{parse.output.repo}}) 已派发审查，复杂度: {{classify.output.level}}"

    # === Review Event 分支 ===

    # 9. 检查是否 merge-ready（2 票 approve）
    - id: check_reviews
      type: http
      config:
        method: GET
        url: "https://api.github.com/repos/{{parse.output.repo}}/pulls/{{parse.output.pr}}/reviews"
        headers:
          Authorization: "Bearer ${GITHUB_TOKEN}"

    # 10. 统计 approve 数
    - id: count_approves
      type: script
      config:
        runtime: javascript
        code: |
          const reviews = input.reviews;
          const latest = {};
          reviews.forEach(r => { latest[r.user.login] = r.state; });
          const approves = Object.values(latest).filter(s => s === 'APPROVED').length;
          return { approves, merge_ready: approves >= 2 };

    # 11. 条件：是否达到 2 票
    - id: check_merge_ready
      type: condition
      config:
        expression: "{{count_approves.output.merge_ready}}"
        branches:
          - value: true
          - value: false

    # 12. 通知 merge-ready + @ 负责人确认
    - id: notify_merge_ready
      type: human_action
      config:
        participant: { type: role, id: "tech_lead" }
        channel: "group:dev_team"
        message: |
          ✅ PR #{{parse.output.pr}} 已获 {{count_approves.output.approves}} 票 approve，可以 merge。
          回复「merge」确认合并
        expect:
          type: keyword
          keywords:
            merge: ["merge", "合并"]
        timeout:
          duration: "8h"
          default: "merge"  # 8h 没回复自动 merge

    # 13. LLM 摘要
    - id: llm_summary
      type: bot_action
      config:
        bot_id: "coda_bot"
        action: "summarize"
        input:
          context: "PR #{{parse.output.pr}} review verdict"
          content: "{{dispatch_review.output.findings}}"

  edges:
    - { from: parse, to: filter }
    - { from: filter, to: route, branch: "需处理" }
    - { from: filter, to: __end__, branch: "跳过" }
    - { from: route, to: fetch_files, branch: "pull_request" }
    - { from: route, to: check_reviews, branch: "pull_request_review" }
    - { from: fetch_files, to: classify }
    - { from: classify, to: post_pending }
    - { from: post_pending, to: dispatch_review }
    - { from: dispatch_review, to: notify_cr }
    - { from: notify_cr, to: llm_summary }
    - { from: check_reviews, to: count_approves }
    - { from: count_approves, to: check_merge_ready }
    - { from: check_merge_ready, to: notify_merge_ready, branch: true }
    - { from: check_merge_ready, to: __end__, branch: false }
```

这个 flow 完整替代了 dispatcher 的 1200 行 Go 代码，而且：
- **cancel-on-push** 用 `concurrency.strategy: cancel_previous` 一行搞定
- **人工确认** 用 human_action 节点，在群里回复即可
- **可视化** 直接在编辑器里看到 DAG 拓扑
- **可修改** 不需要改代码重新编译部署，UI 上拖拽调整

---

## 十二、可观测性

### 12.1 执行面板

```
┌───────────────────────────────────────────────────┐
│  PR Review Dispatcher — 执行历史                    │
├───────────────────────────────────────────────────┤
│                                                   │
│  #1042  ✅ SUCCESS   PR#155 octo-server   2.3s    │
│  #1041  ⏳ WAITING   PR#156 octo-web     (@ Yu)   │
│  #1040  ❌ FAILED    PR#154 adapters     timeout  │
│  #1039  🚫 CANCELLED PR#153 (被 #1040 cancel)     │
│                                                   │
│  ── 点开 #1041 ──                                  │
│                                                   │
│  parse      ✅ 0.1s   {repo: "octo-web", pr: 156} │
│  filter     ✅ 0.0s   → 需处理                     │
│  route      ✅ 0.0s   → pull_request               │
│  fetch_files ✅ 0.8s  3 files, 67 LoC              │
│  classify   ✅ 0.0s   → normal                     │
│  post_pending ✅ 0.3s GitHub status ✓               │
│  dispatch   ✅ 12m    YUJ-2001 verdict: approve    │
│  ask_yu     ⏳ waiting 已 @ Yu, 等待回复…           │
│                                                   │
└───────────────────────────────────────────────────┘
```

### 12.2 实时状态推送

执行状态变更时，通过 Octo IM 推送到指定群/DM：
- 节点失败 → 立即通知
- 人工节点等待超时 → 自动提醒
- 整个 flow 完成 → 发汇总

---

## 十三、路线图

### Phase 1 — 引擎 MVP（4-6 周）

**目标**：能跑通 PR review 流程，替代 Go dispatcher

| 模块 | 内容 |
|---|---|
| Flow Engine | DAG 解析、节点执行、状态管理、context 传递 |
| 触发器 | Webhook + 手动触发 |
| 节点类型 | script + http + condition |
| 并发控制 | cancel_previous |
| 存储 | PostgreSQL（flows + executions + node_executions） |
| API | Flow CRUD + Execute + 执行历史 |
| 可视化 | 只读 DAG 渲染（React Flow 库） |

**验证标准**：用 Octo Flow 表达的 PR review flow 接管 GitHub webhook，dispatcher 可以关掉。

### Phase 2 — 人机混编（4 周）

**目标**：加入人工节点和 Bot 动作节点

| 模块 | 内容 |
|---|---|
| human_action 节点 | 发消息 + wait handle + 超时提醒 + 升级 |
| bot_action 节点 | Bot Capability Registry + Bot 调用协议 |
| 角色系统 | Space 角色 + 动态解析 |
| 消息触发器 | 关键词 / 命令 / 正则匹配 |
| 可视化编辑器 | 拖拽建 flow（React Flow + 节点面板） |

### Phase 3 — 自然语言 + 模板市场（6 周）

**目标**：非技术用户也能建 flow

| 模块 | 内容 |
|---|---|
| NL→Flow | LLM 翻译层 + Schema 校验 + 预览 |
| NL 编辑 | 用自然语言修改已有 flow |
| 模板市场 | 发布 / 搜索 / 安装 / 评分 |
| 内置模板 | 8 个官方模板 |
| Intent 触发器 | LLM 判断消息意图触发 flow |

### Phase 4 — 生态扩展（持续）

- 更多集成（GitLab / Jira / Feishu / 钉钉）
- flow 间调用（subflow）
- 并行节点 + 循环节点
- flow 权限管理（谁能看 / 谁能编辑 / 谁能执行）
- 审计日志
- flow 导入/导出（跨 Space 迁移）

---

## 十四、与现有系统的关系

| 现有组件 | Octo Flow 后的定位 |
|---|---|
| reviewbot-dispatcher (Go) | Phase 1 完成后**下线**，flow 替代 |
| echo-webhook-receiver | 合并为 flow 里的一个节点 |
| multica daemon | 保留，flow 通过 bot_action 节点调用 |
| Windmill / Kestra（已部署） | 作为调研参考，不进生产，Octo Flow 是自研 |

---

## 十五、技术选型建议

| 维度 | 建议 |
|---|---|
| 引擎语言 | Go（与 octo-server 一致，性能好） |
| Flow 定义格式 | YAML 存储 + JSONB 查询（两种视图） |
| 可视化库 | React Flow（最成熟的 DAG 编辑器，MIT） |
| LLM 调用 | octo-server 已有的 LLM 网关 |
| 定时器 | Go cron library + DB 持久化（不依赖 OS cron） |
| 消息拦截 | octo-server 消息管道加 middleware |

---

> 这份设计文档覆盖了架构、数据模型、API、节点类型、人机混编、自然语言编排、模板市场、完整示例和路线图。可以直接作为 RFC 评审和派单的基础。
