function showNotification(message, type) {
    if (!type) { type = 'success'; }
    const container = document.getElementById('notification-container');
    const notification = document.createElement('div');
    const bgColor = (type === 'success') ? 'bg-green-100 dark:bg-green-900/30 border-green-400 dark:border-green-600' : 'bg-red-100 dark:bg-red-900/30 border-red-400 dark:border-red-600';
    const textColor = (type === 'success') ? 'text-green-800 dark:text-green-200' : 'text-red-800 dark:text-red-200';
    const icon = (type === 'success') ? 'fa-check-circle' : 'fa-exclamation-circle';
    notification.className = 'notification-enter ' + bgColor + ' ' + textColor + ' border-l-4 p-4 mb-3 rounded-lg shadow-lg flex items-center justify-between max-w-2xl mx-auto';
    const contentDiv = document.createElement('div');
    contentDiv.className = 'flex items-center space-x-3';
    const iconEl = document.createElement('i');
    iconEl.className = 'fas ' + icon + ' text-xl';
    const spanEl = document.createElement('span');
    spanEl.className = 'font-medium';
    spanEl.textContent = message;
    contentDiv.appendChild(iconEl);
    contentDiv.appendChild(spanEl);
    const closeButton = document.createElement('button');
    closeButton.className = 'ml-4 ' + textColor + ' hover:opacity-75 transition';
    const closeIcon = document.createElement('i');
    closeIcon.className = 'fas fa-times';
    closeButton.appendChild(closeIcon);
    closeButton.addEventListener('click', function() {
        notification.classList.remove('notification-enter');
        notification.classList.add('notification-exit');
        setTimeout(function(){ notification.remove(); }, 300);
    });
    notification.appendChild(contentDiv);
    notification.appendChild(closeButton);
    container.appendChild(notification);
    setTimeout(function(){
        notification.classList.remove('notification-enter');
        notification.classList.add('notification-exit');
        setTimeout(function(){ notification.remove(); }, 300);
    }, 5000);
}

function queueNotification(message, type) {
    if (!type) { type = 'success'; }
    sessionStorage.setItem('pendingNotification', JSON.stringify({ message: message, type: type }));
}

window.addEventListener('DOMContentLoaded', function(){
    // Read notification headers once per request (avoids duplicates with OOB swaps)
    // De-dup notifications and avoid double showing across OOB swaps
    window.__htmxNotifSeen = window.__htmxNotifSeen || {};
    document.body.addEventListener('htmx:afterRequest', function(evt){
        try {
            const xhr = (evt.detail && evt.detail.xhr) ? evt.detail.xhr : null;
            if (xhr) {
                let notification = null;
                // Prefer Base64 header for safe non-ASCII transport
                const b64 = xhr.getResponseHeader('X-Notification-Base64');
                if (b64) {
                    try {
                        // Decode base64 into UTF-8 string
                        const binStr = atob(b64);
                        if (window.TextDecoder) {
                            const len = binStr.length;
                            const bytes = new Uint8Array(len);
                            for (let i = 0; i < len; i++) bytes[i] = binStr.charCodeAt(i);
                            notification = new TextDecoder('utf-8').decode(bytes);
                        } else {
                            // Fallback for very old browsers
                            notification = decodeURIComponent(escape(binStr));
                        }
                    } catch (e) {
                        // Fallback to plain header if decoding fails
                        notification = xhr.getResponseHeader('X-Notification');
                    }
                } else {
                    notification = xhr.getResponseHeader('X-Notification');
                }
                const notificationType = xhr.getResponseHeader('X-Notification-Type');
                if (notification) {
                    const key = (xhr.responseURL || '') + '|' + (notificationType || 'success') + '|' + notification;
                    if (!window.__htmxNotifSeen[key]) {
                        window.__htmxNotifSeen[key] = Date.now();
                        // Clean up old entries after 10s
                        setTimeout(function(){ delete window.__htmxNotifSeen[key]; }, 10000);
                        showNotification(notification, notificationType || 'success');
                    }
                }
            }
        } catch (e) {}
    });
    document.body.addEventListener('htmx:responseError', function(evt){
        try {
            const xhr = (evt.detail && evt.detail.xhr) ? evt.detail.xhr : null;
            let msg = 'Erro na operação';
            if (xhr && xhr.responseText) { msg = xhr.responseText; }
            showNotification(msg, 'error');
        } catch (e) {}
    });
    // Close modal and reset form after successful topic creation
    document.body.addEventListener('topic-created', function(){
        try {
            const modal = document.getElementById('createTopicModal');
            if (modal) { modal.classList.add('hidden'); }
            const form = document.getElementById('createTopicForm');
            if (form) { form.reset(); }
        } catch (e) {}
    });
    const pending = sessionStorage.getItem('pendingNotification');
    if (pending) {
        try {
            const parsed = JSON.parse(pending);
            sessionStorage.removeItem('pendingNotification');
            const msg = (parsed && parsed.message) ? parsed.message : '';
            const tp = (parsed && parsed.type) ? parsed.type : 'success';
            if (msg) { showNotification(msg, tp); }
        } catch (e) { sessionStorage.removeItem('pendingNotification'); }
    }
});
