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
    function filterItems(input) {
        const query = (input.value || '').trim().toLowerCase();
        const targetTableId = input.getAttribute('data-filter-target');
        if (!targetTableId) return;

        const table = document.getElementById(targetTableId);
        if (!table) return;

        const rows = table.querySelectorAll('tbody tr[data-filter-value]');
        rows.forEach(function (row) {
            const value = (row.getAttribute('data-filter-value') || '').toLowerCase();
            const show = query === '' || value.includes(query);
            row.style.display = show ? '' : 'none';
        });
    }

    function setupFilters() {
        const inputs = document.querySelectorAll('input[data-filter-target]');
        inputs.forEach(function (input) {
            // Remove existing listener to avoid duplicates if setupFilters is called multiple times
            input.removeEventListener('input', handleFilterInput);
            input.addEventListener('input', handleFilterInput);
            filterItems(input);
        });
    }

    function handleFilterInput(e) {
        filterItems(e.target);
    }

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', setupFilters);
    } else {
        setupFilters();
    }

    document.addEventListener('htmx:afterSwap', function (evt) {
        const target = evt && evt.detail && evt.detail.target;
        if (!target) return;
        
        // Se o swap trouxe uma nova tabela ou o próprio container da lista, re-configura os filtros
        setupFilters();
    });
})();

function switchTab(tabId) {
    // Hide all tab contents with transition
    const tabContents = document.querySelectorAll('.tab-content');
    tabContents.forEach(content => {
        content.classList.add('hidden');
        content.classList.remove('fade-in');
    });

    // Reset all tab buttons
    const tabButtons = document.querySelectorAll('.tab-button');
    tabButtons.forEach(button => {
        button.classList.remove('tab-button-active');
        button.classList.add('text-neutral-600', 'dark:text-neutral-400');
        button.classList.remove('text-guara-600', 'dark:text-guara-400', 'border-guara-500');
        button.classList.add('border-transparent');
    });

    // Show selected tab with animation
    const selectedTab = document.getElementById(tabId);
    if (selectedTab) {
        selectedTab.classList.remove('hidden');
        selectedTab.classList.add('fade-in');
    }

    // Activate selected button
    const selectedButton = document.querySelector(`[data-tab="${tabId}"]`);
    if (selectedButton) {
        selectedButton.classList.add('tab-button-active');
        selectedButton.classList.remove('text-neutral-600', 'dark:text-neutral-400', 'border-transparent');
        selectedButton.classList.add('text-guara-600', 'dark:text-guara-400', 'border-guara-500');
    }
}
