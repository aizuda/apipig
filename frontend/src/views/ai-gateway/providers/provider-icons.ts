import ai21Icon from '@lobehub/icons-static-svg/icons/ai21-brand-color.svg'
import anthropicIcon from '@lobehub/icons-static-svg/icons/anthropic.svg'
import azureAiIcon from '@lobehub/icons-static-svg/icons/azureai-color.svg'
import baiduIcon from '@lobehub/icons-static-svg/icons/baiducloud-color.svg'
import bedrockIcon from '@lobehub/icons-static-svg/icons/bedrock-color.svg'
import bytedanceIcon from '@lobehub/icons-static-svg/icons/bytedance-color.svg'
import cerebrasIcon from '@lobehub/icons-static-svg/icons/cerebras-color.svg'
import cloudflareIcon from '@lobehub/icons-static-svg/icons/cloudflare-color.svg'
import cohereIcon from '@lobehub/icons-static-svg/icons/cohere-color.svg'
import deepseekIcon from '@lobehub/icons-static-svg/icons/deepseek-color.svg'
import geminiIcon from '@lobehub/icons-static-svg/icons/gemini-color.svg'
import groqIcon from '@lobehub/icons-static-svg/icons/groq.svg'
import huggingFaceIcon from '@lobehub/icons-static-svg/icons/huggingface-color.svg'
import metaIcon from '@lobehub/icons-static-svg/icons/metaai-color.svg'
import minimaxIcon from '@lobehub/icons-static-svg/icons/minimax-color.svg'
import mistralIcon from '@lobehub/icons-static-svg/icons/mistral-color.svg'
import moonshotIcon from '@lobehub/icons-static-svg/icons/moonshot.svg'
import nvidiaIcon from '@lobehub/icons-static-svg/icons/nvidia-color.svg'
import ollamaIcon from '@lobehub/icons-static-svg/icons/ollama.svg'
import openaiIcon from '@lobehub/icons-static-svg/icons/openai.svg'
import openrouterIcon from '@lobehub/icons-static-svg/icons/openrouter-color.svg'
import perplexityIcon from '@lobehub/icons-static-svg/icons/perplexity-color.svg'
import qwenIcon from '@lobehub/icons-static-svg/icons/qwen-color.svg'
import replicateIcon from '@lobehub/icons-static-svg/icons/replicate.svg'
import siliconCloudIcon from '@lobehub/icons-static-svg/icons/siliconcloud-color.svg'
import tencentIcon from '@lobehub/icons-static-svg/icons/tencentcloud-color.svg'
import togetherIcon from '@lobehub/icons-static-svg/icons/together-color.svg'
import xaiIcon from '@lobehub/icons-static-svg/icons/xai.svg'
import zhipuIcon from '@lobehub/icons-static-svg/icons/zhipu-color.svg'

export interface ProviderIconDefinition {
  id: string
  name: string
  source: string
  keywords: string[]
}

export const providerIcons: ProviderIconDefinition[] = [
  { id: 'openai', name: 'OpenAI', source: openaiIcon, keywords: ['GPT', 'ChatGPT'] },
  { id: 'anthropic', name: 'Anthropic', source: anthropicIcon, keywords: ['Claude'] },
  { id: 'gemini', name: 'Google Gemini', source: geminiIcon, keywords: ['Google', 'Vertex AI'] },
  { id: 'xai', name: 'xAI', source: xaiIcon, keywords: ['Grok'] },
  { id: 'deepseek', name: 'DeepSeek', source: deepseekIcon, keywords: ['深度求索'] },
  { id: 'qwen', name: '通义千问', source: qwenIcon, keywords: ['Qwen', 'DashScope', '阿里云'] },
  { id: 'mistral', name: 'Mistral AI', source: mistralIcon, keywords: ['Mixtral'] },
  { id: 'cohere', name: 'Cohere', source: cohereIcon, keywords: ['Command'] },
  { id: 'meta', name: 'Meta AI', source: metaIcon, keywords: ['Llama'] },
  { id: 'azureai', name: 'Azure AI', source: azureAiIcon, keywords: ['Microsoft'] },
  { id: 'bedrock', name: 'Amazon Bedrock', source: bedrockIcon, keywords: ['AWS'] },
  {
    id: 'huggingface',
    name: 'Hugging Face',
    source: huggingFaceIcon,
    keywords: ['Inference'],
  },
  { id: 'groq', name: 'Groq', source: groqIcon, keywords: ['LPU'] },
  { id: 'ollama', name: 'Ollama', source: ollamaIcon, keywords: ['Local'] },
  { id: 'nvidia', name: 'NVIDIA NIM', source: nvidiaIcon, keywords: ['NIM', 'NeMo'] },
  { id: 'perplexity', name: 'Perplexity', source: perplexityIcon, keywords: ['Sonar'] },
  { id: 'moonshot', name: 'Moonshot AI', source: moonshotIcon, keywords: ['Kimi', '月之暗面'] },
  { id: 'zhipu', name: '智谱 AI', source: zhipuIcon, keywords: ['GLM', 'BigModel'] },
  { id: 'minimax', name: 'MiniMax', source: minimaxIcon, keywords: ['海螺'] },
  { id: 'baidu', name: '百度智能云', source: baiduIcon, keywords: ['文心', 'ERNIE'] },
  { id: 'bytedance', name: '火山引擎', source: bytedanceIcon, keywords: ['豆包', 'ByteDance'] },
  { id: 'tencent', name: '腾讯云', source: tencentIcon, keywords: ['混元', 'Hunyuan'] },
  { id: 'cloudflare', name: 'Cloudflare AI', source: cloudflareIcon, keywords: ['Workers AI'] },
  { id: 'openrouter', name: 'OpenRouter', source: openrouterIcon, keywords: ['Router'] },
  { id: 'together', name: 'Together AI', source: togetherIcon, keywords: ['Together'] },
  { id: 'cerebras', name: 'Cerebras', source: cerebrasIcon, keywords: ['Inference'] },
  { id: 'replicate', name: 'Replicate', source: replicateIcon, keywords: ['Models'] },
  { id: 'ai21', name: 'AI21 Labs', source: ai21Icon, keywords: ['Jamba'] },
  {
    id: 'siliconcloud',
    name: 'SiliconCloud',
    source: siliconCloudIcon,
    keywords: ['SiliconFlow', '硅基流动'],
  },
]

const iconsByID = new Map(providerIcons.map((icon) => [icon.id, icon]))
const iconAliases: Record<string, string> = {
  amazon: 'bedrock',
  aliyun: 'qwen',
  alibaba: 'qwen',
  aws: 'bedrock',
  azure: 'azureai',
  chatgpt: 'openai',
  claude: 'anthropic',
  dashscope: 'qwen',
  doubao: 'bytedance',
  ernie: 'baidu',
  glm: 'zhipu',
  google: 'gemini',
  grok: 'xai',
  hunyuan: 'tencent',
  kimi: 'moonshot',
  llama: 'meta',
  microsoft: 'azureai',
  siliconflow: 'siliconcloud',
  vertex: 'gemini',
  wenxin: 'baidu',
}

function normalizeIconKey(value?: string) {
  return (value || '').trim().toLowerCase().replaceAll('_', '-')
}

export function getProviderIcon(value?: string) {
  return iconsByID.get(normalizeIconKey(value))
}

export function resolveProviderIcon(...values: Array<string | undefined>) {
  for (const value of values) {
    const normalized = normalizeIconKey(value)
    if (!normalized) continue
    const direct = iconsByID.get(normalized)
    if (direct) return direct
    const alias = iconAliases[normalized]
    if (alias) return iconsByID.get(alias)
    for (const [candidate, iconID] of Object.entries(iconAliases)) {
      if (candidate.length >= 3 && normalized.includes(candidate)) return iconsByID.get(iconID)
    }
  }
}
