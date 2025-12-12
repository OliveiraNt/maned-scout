tailwind.config = {
    darkMode: 'class',
}
if (localStorage.getItem('darkMode') === 'true' || (!localStorage.getItem('darkMode') && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
    document.documentElement.classList.add('dark');
}

(function () {
    function filterTopics(query) {
        const q = (query || '').trim().toLowerCase();
        const rows = document.querySelectorAll('#topicsTable tbody tr[data-topic-name]');
        if (!rows.length) return;
        rows.forEach(function (row) {
            const name = (row.getAttribute('data-topic-name') || '').toLowerCase();
            const show = q === '' || name.includes(q);
            row.style.display = show ? '' : 'none';
        });
    }

    function setupFilter() {
        const input = document.querySelector('input[type="text"][placeholder="Buscar tópicos"]');
        if (!input) return;
        input.addEventListener('input', function (e) {
            filterTopics(e.target.value);
        });
        filterTopics(input.value);
    }

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', setupFilter);
    } else {
        setupFilter();
    }

    document.addEventListener('htmx:afterSwap', function (evt) {
        const target = evt && evt.detail && evt.detail.target;
        if (!target) return;
        if (target.id === 'topics-list' || target.querySelector && target.querySelector('#topicsTable')) {
            const input = document.querySelector('input[type="text"][placeholder="Buscar tópicos"]');
            if (input) filterTopics(input.value);
        }
    });
})();
