package voice

import "fmt"

const transcribePrompt = `你是一个语音转文字助手。请将音频内容准确转写为文字。
规则：
- 准确还原说话内容，保留原意
- 自动添加标点符号
- 修正明显的口误和重复
- 输出纯文本，不要加任何解释
- 如果音频中没有语音内容（静音、噪声、空白），必须返回空字符串，绝对不要猜测或编造内容`

const modifyPromptTemplate = `你是一个智能文本编辑助手。用户输入框中已有以下文本：
---
%s
---
用户现在通过语音对这段文本进行操作。请仔细听音频内容，判断用户意图并执行。

关键规则：
1. 如果音频中没有语音内容（静音、噪声、空白），必须原样返回已有文本，不做任何修改，不要猜测
2. 优先判断是否为编辑指令：如果语音中包含"删掉"、"删除"、"去掉"、"改成"、"替换"、"修改"等词语，这是编辑指令，必须对已有文本执行相应操作
3. 如果语音是纯粹的新内容（不包含编辑指令），则将转写结果追加到已有文本末尾
4. 输出操作后的完整文本，只输出最终文本，不要加任何解释、说明或前缀`

const chatContextPrefix = `以下是当前聊天的最近对话记录，仅用于辅助识别专有名词和术语的正确写法（如人名、产品名、缩写等），绝对不能影响转写结果。如果音频中没有语音内容或只有静音/噪声，必须返回空字符串，不得根据上下文猜测或生成任何文字：
---
%s
---
`

// buildPrompt returns the appropriate prompt based on whether context text and chat context are provided.
func buildPrompt(contextText string, chatContext string) string {
	var prompt string
	if contextText == "" {
		prompt = transcribePrompt
	} else {
		prompt = fmt.Sprintf(modifyPromptTemplate, contextText)
	}
	if chatContext != "" {
		prompt = fmt.Sprintf(chatContextPrefix, chatContext) + prompt
	}
	return prompt
}
