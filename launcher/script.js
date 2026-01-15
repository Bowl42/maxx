/**
 * Maxx 启动画面脚本
 * 负责调用后端检查服务状态，后端返回跳转地址后执行跳转
 */
(function() {
    'use strict';

    // 配置
    const CONFIG = {
        checkInterval: 500,      // 检查间隔 (ms)
        maxWaitTime: 60000,      // 最大等待时间 (ms)
        redirectDelay: 300       // 重定向前的延迟 (ms)
    };

    // DOM 元素
    const elements = {
        statusContainer: document.getElementById('status-container'),
        statusText: document.getElementById('status-text'),
        loadingSpinner: document.getElementById('loading-spinner'),
        errorContainer: document.getElementById('error-container'),
        errorMessage: document.getElementById('error-message'),
        retryButton: document.getElementById('retry-button'),
        quitButton: document.getElementById('quit-button'),
        versionText: document.getElementById('version-text')
    };

    // 状态
    let checkTimer = null;
    let startTime = Date.now();

    /**
     * 更新状态文本
     */
    function updateStatus(text) {
        elements.statusText.textContent = text;
    }

    /**
     * 显示错误
     */
    function showError(message) {
        elements.statusContainer.style.display = 'none';
        elements.errorContainer.classList.remove('hidden');
        elements.errorMessage.textContent = message;
    }

    /**
     * 隐藏错误，显示加载状态
     */
    function hideError() {
        elements.errorContainer.classList.add('hidden');
        elements.statusContainer.style.display = 'flex';
        elements.statusContainer.classList.remove('success');
    }

    /**
     * 显示成功状态
     */
    function showSuccess(message) {
        elements.statusContainer.classList.add('success');
        updateStatus(message || '启动完成，正在跳转...');
    }

    /**
     * 重定向到指定地址
     */
    function redirectTo(url) {
        showSuccess('启动完成，正在跳转...');
        setTimeout(() => {
            window.location.href = url;
        }, CONFIG.redirectDelay);
    }

    /**
     * 调用后端检查服务器状态
     */
    async function checkServer() {
        const elapsed = Date.now() - startTime;

        // 超时检查
        if (elapsed > CONFIG.maxWaitTime) {
            clearInterval(checkTimer);
            showError('服务器启动超时\n\n请检查日志文件或重试启动。');
            return;
        }

        // 更新状态文本（显示等待时间）
        const seconds = Math.floor(elapsed / 1000);
        if (seconds > 0) {
            updateStatus(`正在启动服务... (${seconds}s)`);
        }

        // 调用后端函数检查状态
        try {
            if (!window.go || !window.go.desktop || !window.go.desktop.LauncherApp) {
                console.log('[Launcher] Waiting for Wails runtime...');
                return;
            }

            const status = await window.go.desktop.LauncherApp.CheckServerStatus();

            // 更新状态消息
            if (status.Message) {
                updateStatus(status.Message);
            }

            // 检查是否需要跳转
            if (status.Ready && status.RedirectURL) {
                clearInterval(checkTimer);
                redirectTo(status.RedirectURL);
                return;
            }

            // 检查是否有错误
            if (status.Error) {
                clearInterval(checkTimer);
                showError(status.Error);
                return;
            }

            // 继续等待...
        } catch (err) {
            console.error('[Launcher] Check status failed:', err);
            // 继续等待，可能是 Wails 还没准备好
        }
    }

    /**
     * 重试启动
     */
    async function retry() {
        hideError();
        startTime = Date.now();
        updateStatus('正在重新启动服务...');

        // 调用后端重启服务器
        try {
            if (window.go && window.go.desktop && window.go.desktop.LauncherApp) {
                await window.go.desktop.LauncherApp.RestartServer();
            }
        } catch (err) {
            console.error('[Launcher] Restart failed:', err);
            showError('重启服务器失败: ' + (err.message || err));
            return;
        }

        // 开始检查
        checkTimer = setInterval(checkServer, CONFIG.checkInterval);
        checkServer();
    }

    /**
     * 退出应用
     */
    function quit() {
        if (window.go && window.go.desktop && window.go.desktop.LauncherApp) {
            window.go.desktop.LauncherApp.Quit();
        } else {
            window.close();
        }
    }

    /**
     * 获取版本信息
     */
    async function loadVersion() {
        // 等待 Wails 准备好
        const maxWait = 5000;
        const startWait = Date.now();

        while (Date.now() - startWait < maxWait) {
            if (window.go && window.go.desktop && window.go.desktop.LauncherApp) {
                try {
                    const version = await window.go.desktop.LauncherApp.GetVersion();
                    if (version) {
                        elements.versionText.textContent = version;
                    }
                } catch (err) {
                    console.error('[Launcher] Failed to get version:', err);
                }
                return;
            }
            await new Promise(resolve => setTimeout(resolve, 100));
        }
    }

    /**
     * 初始化
     */
    function init() {
        console.log('[Launcher] Initializing...');

        // 绑定事件
        elements.retryButton.addEventListener('click', retry);
        elements.quitButton.addEventListener('click', quit);

        // 加载版本
        loadVersion();

        // 开始检查服务器状态
        checkTimer = setInterval(checkServer, CONFIG.checkInterval);
        checkServer();
    }

    // 等待 DOM 准备就绪
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
})();
