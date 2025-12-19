tailwind.config = {
    darkMode: 'class',
    theme: {
        extend: {
            colors: {
                neutral: {
                    50:  '#F8F9FA',
                    100: '#F1F3F5',
                    200: '#E9ECEF',
                    300: '#DEE2E6',
                    400: '#CED4DA',
                    500: '#ADB5BD',
                    600: '#6C757D',
                    700: '#495057',
                    800: '#343A40',
                    900: '#212529',
                },
                guara: {
                    50:  '#FFF4EC',
                    100: '#FFE3D1',
                    200: '#FFC6A3',
                    300: '#FFA066',
                    400: '#FF7A29',
                    500: '#E85D04',
                    600: '#C94A02',
                    700: '#A23A02',
                    800: '#7A2B01',
                    900: '#4F1C00',
                },
                success: '#2FBF71',
                warning: '#F4A261',
                error:   '#E63946',
                info:    '#3A86FF',
                surface: {
                    dark: '#161A22',
                    darker: '#0F1115',
                },
            },
            fontFamily: {
                sans: ['Inter', 'system-ui', 'sans-serif'],
                display: ['Space Grotesk', 'Inter', 'sans-serif'],
            },
            borderRadius: {
                lg: '0.5rem',
                xl: '0.75rem',
            },
        },
    },
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
