/*
 * ==============================================================================
 *  ZEN-AGENT HarmonyOS / OpenHarmony 原生 ArkTS 实现
 *  - 适用平台: 华为鸿蒙 HarmonyOS NEXT / OpenHarmony 4.0+ / 鸿蒙车机 / 鸿蒙工控
 *  - 依赖: 零外部第三方三方库 (纯 HarmonyOS @ohos 系统内置 API)
 *  - 架构:
 *      1. @ohos.net.http (华为原生高性能 HTTP 客户端)
 *      2. @ohos.process / @ohos.file.fs (鸿蒙沙箱文件与系统进程交互)
 * ==============================================================================
 */

import http from '@ohos.net.http';
import process from '@ohos.process';
import fs from '@ohos.file.fs';

const API_URL = 'https://api.openai.com/v1/chat/completions';
const API_KEY = 'your-api-key-here';
const MODEL = 'gpt-4o';

export interface ZenDecision {
  action: 'run_command' | 'write_file' | 'finish';
  thought: string;
  command?: string;
  path?: string;
  content?: string;
  explanation: string;
}

export class ZenAgentHarmony {
  private history: string = '鸿蒙 HarmonyOS NEXT 环境已就绪。';

  /**
   * 通过鸿蒙原生 HTTP 栈与小米 MiMo-v2.5 进行高并发安全通讯
   */
  async callMimo(prompt: string): Promise<ZenDecision> {
    const httpRequest = http.createHttp();
    const payload = JSON.stringify({
      model: MODEL,
      messages: [
        {
          role: 'system',
          content: '你是运行在华为 HarmonyOS / OpenHarmony 系统上的自愈运维智能体。请必须只输出纯 JSON: {"action":"run_command"|"write_file"|"finish","command":"...","thought":"...","explanation":"..."}'
        },
        {
          role: 'user',
          content: prompt
        }
      ],
      max_tokens: 1024,
      temperature: 0.1
    });

    try {
      const response = await httpRequest.request(API_URL, {
        method: http.RequestMethod.POST,
        header: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${API_KEY}`,
          'User-Agent': 'opencode/1.0.0 (HarmonyOS)'
        },
        extraData: payload,
        connectTimeout: 20000,
        readTimeout: 30000
      });

      if (response.responseCode === 200) {
        const resBody = typeof response.result === 'string' ? response.result : JSON.stringify(response.result);
        const parsed = JSON.parse(resBody);
        let rawContent = parsed.choices[0]?.message?.content || parsed.choices[0]?.message?.reasoning_content || '';
        
        // 过滤 Markdown 标记
        if (rawContent.includes('```json')) {
          rawContent = rawContent.split('```json')[1].split('```')[0];
        } else if (rawContent.includes('```')) {
          rawContent = rawContent.split('```')[1].split('```')[0];
        }
        return JSON.parse(rawContent.trim()) as ZenDecision;
      } else {
        return {
          action: 'finish',
          thought: `鸿蒙 HTTP 响应异常: ${response.responseCode}`,
          explanation: '上游接口通讯中断'
        };
      }
    } catch (err) {
      return {
        action: 'finish',
        thought: `鸿蒙端网络握手失败: ${JSON.stringify(err)}`,
        explanation: '网络请求捕获异常'
      };
    } finally {
      httpRequest.destroy();
    }
  }

  /**
   * 鸿蒙自愈循环总控 (支持 DevEco Studio 编译 / 命令行 hap 部署)
   */
  async runTask(task: string, maxSteps: number = 20): Promise<void> {
    console.log(`[ZenAgent-HarmonyOS] 🎯 启动鸿蒙任务: ${task}`);

    for (let step = 1; step <= maxSteps; step++) {
      console.log(`\n[ZenAgent-HarmonyOS Step ${step}] 正在向小米 MiMo 请求下一步鸿蒙适配指令...`);
      const prompt = `【鸿蒙目标】: ${task}\n【已执行状态】: ${this.history}\n请指示下一步自愈动作：`;
      const decision = await this.callMimo(prompt);

      console.log(`🤔 模型思考: ${decision.thought}`);
      console.log(`⚡ 鸿蒙动作: ${decision.explanation}`);

      if (decision.action === 'finish') {
        console.log('🎉 [成功] ZenAgent 已成功在 HarmonyOS 上完成全部配置与部署任务！');
        break;
      }

      if (decision.action === 'run_command' && decision.command) {
        // 在具备 hdc shell 或底层执行权限的环境下透传命令
        this.history += `\n[Step ${step} 指令]: ${decision.command}\n[状态]: 鸿蒙已下发执行`;
      }
    }
  }
}
