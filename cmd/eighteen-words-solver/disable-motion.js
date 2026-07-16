(() => {
    const style = document.createElement("style");
    style.textContent = "*,*::before,*::after{animation:none!important;transition:none!important;scroll-behavior:auto!important}#confetti{display:none!important}";
    document.head.appendChild(style);

    const nativeSetTimeout = window.setTimeout.bind(window);
    window.setTimeout = (callback, delay, ...args) => nativeSetTimeout(callback, delay >= 10000 ? delay : 0, ...args);
})()
