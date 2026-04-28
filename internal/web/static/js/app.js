// API base URL
const API_BASE = '/api';

const calculateForm = document.getElementById('calculate-form');
const addPackForm = document.getElementById('add-pack-form');
const resultDiv = document.getElementById('result');
const errorDiv = document.getElementById('error');
const packErrorDiv = document.getElementById('pack-error');
const packSizesList = document.getElementById('pack-sizes-list');

// Calculate packs
calculateForm?.addEventListener('submit', async (e) => {
    e.preventDefault();

    const order = parseInt(document.getElementById('order').value);
    if (!order || order <= 0) {
        showError('Please enter a valid order quantity');
        return;
    }

    try {
        const response = await fetch(`${API_BASE}/calculate`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ order })
        });

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Calculation failed');
        }

        const result = await response.json();
        displayResult(result);
        hideError();
    } catch (err) {
        showError(err.message);
        hideResult();
    }
});

// Add pack size
addPackForm?.addEventListener('submit', async (e) => {
    e.preventDefault();

    const size = parseInt(document.getElementById('new-size').value);
    if (!size || size <= 0) {
        showPackError('Please enter a valid pack size');
        return;
    }

    try {
        const response = await fetch(`${API_BASE}/pack-sizes`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ size })
        });

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Failed to add pack size');
        }

        document.getElementById('new-size').value = '';
        await loadPackSizes();
        hidePackError();
    } catch (err) {
        showPackError(err.message);
    }
});

// Delete pack size
packSizesList?.addEventListener('click', async (e) => {
    if (e.target.classList.contains('delete-btn')) {
        const size = e.target.dataset.size;

        try {
            const response = await fetch(`${API_BASE}/pack-sizes/${size}`, {
                method: 'DELETE'
            });

            if (!response.ok) {
                const error = await response.json();
                throw new Error(error.error || 'Failed to delete pack size');
            }

            await loadPackSizes();
            hidePackError();
        } catch (err) {
            showPackError(err.message);
        }
    }
});

// Load pack sizes
async function loadPackSizes() {
    try {
        const response = await fetch(`${API_BASE}/pack-sizes`);
        const sizes = await response.json();

        packSizesList.innerHTML = sizes.map(size => `
            <div class="size-item" data-size="${size}">
                <span class="size-value">${size.toLocaleString()} items</span>
                <button class="delete-btn" data-size="${size}">×</button>
            </div>
        `).join('');
    } catch (err) {
        console.error('Failed to load pack sizes:', err);
    }
}

// Display result
function displayResult(result) {
    document.getElementById('result-order').textContent = result.order.toLocaleString();
    document.getElementById('result-total-items').textContent = result.total_items.toLocaleString();
    document.getElementById('result-total-packs').textContent = result.total_packs;

    const breakdown = document.getElementById('pack-breakdown');
    breakdown.innerHTML = Object.entries(result.packs)
        .sort((a, b) => b[0] - a[0])
        .map(([size, count]) => `
            <div class="pack-item">
                <span class="pack-size">${parseInt(size).toLocaleString()} items</span>
                <span class="pack-count">× ${count}</span>
            </div>
        `).join('');

    resultDiv.classList.remove('hidden');
}

function hideResult() {
    resultDiv.classList.add('hidden');
}

function showError(message) {
    errorDiv.textContent = message;
    errorDiv.classList.remove('hidden');
}

function hideError() {
    errorDiv.classList.add('hidden');
}

function showPackError(message) {
    packErrorDiv.textContent = message;
    packErrorDiv.classList.remove('hidden');
}

function hidePackError() {
    packErrorDiv.classList.add('hidden');
}
