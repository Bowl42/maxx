/**
 * Transport 工厂函数
 *
 * 由于桌面客户端现在通过 HTTP Server 提供前端，
 * 所有环境都使用 HttpTransport
 */

import type { Transport, TransportType, TransportConfig } from './interface';
import { HttpTransport } from './http-transport';

/**
 * Transport 初始化状态
 */
type TransportState =
  | { status: 'uninitialized' }
  | { status: 'initializing' }
  | { status: 'ready'; transport: Transport; type: TransportType }
  | { status: 'error'; error: Error };

let state: TransportState = { status: 'uninitialized' };
let initPromise: Promise<Transport> | null = null;

/**
 * 检测当前运行环境
 * 现在所有环境都使用 HTTP
 */
export function detectTransportType(): TransportType {
  return 'http';
}


/**
 * 初始化全局 Transport 单例（异步）
 * 在应用启动时调用一次
 */
export async function initializeTransport(config?: TransportConfig): Promise<Transport> {
  // 已经初始化完成
  if (state.status === 'ready') {
    return state.transport;
  }

  // 正在初始化中，等待完成
  if (state.status === 'initializing' && initPromise) {
    return initPromise;
  }

  // 初始化失败过
  if (state.status === 'error') {
    throw state.error;
  }

  // 开始初始化
  state = { status: 'initializing' };
  console.log('[Transport] Initializing HttpTransport...');

  initPromise = Promise.resolve()
    .then(() => {
      const transport = new HttpTransport(config);
      state = { status: 'ready', transport, type: 'http' };
      console.log('[Transport] Ready: HttpTransport');
      return transport;
    })
    .catch((error) => {
      state = { status: 'error', error };
      console.error('[Transport] Initialization failed:', error);
      throw error;
    });

  return initPromise;
}

/**
 * 获取全局 Transport 单例
 *
 * 重要：必须先调用 initializeTransport() 完成初始化
 * 在 React 组件中，使用 useTransport() hook 来获取 transport
 *
 * @throws Error 如果 transport 未初始化
 */
export function getTransport(): Transport {
  if (state.status === 'ready') {
    return state.transport;
  }

  if (state.status === 'uninitialized') {
    throw new Error(
      '[Transport] Transport not initialized. Call initializeTransport() first, or use useTransport() hook.'
    );
  }

  if (state.status === 'initializing') {
    throw new Error(
      '[Transport] Transport is still initializing. Use useTransport() hook or await initializeTransport().'
    );
  }

  if (state.status === 'error') {
    throw state.error;
  }

  // This should never happen
  throw new Error('[Transport] Unknown transport state');
}

/**
 * 获取当前 Transport 初始化状态
 * 用于 TransportProvider 等需要检查状态的场景
 */
export function getTransportState(): TransportState {
  return state;
}

/**
 * 检查 Transport 是否已初始化完成
 */
export function isTransportReady(): boolean {
  return state.status === 'ready';
}

/**
 * 获取 Transport 类型
 * 仅在初始化完成后有效
 */
export function getTransportType(): TransportType | null {
  if (state.status === 'ready') {
    return state.type;
  }
  return null;
}

/**
 * 重置 Transport 单例（用于测试）
 */
export function resetTransport(): void {
  if (state.status === 'ready') {
    state.transport.disconnect();
  }
  state = { status: 'uninitialized' };
  initPromise = null;
}
