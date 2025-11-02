// Preset configurations
const presets = {
    standard: {
        packSizes: '250, 500, 1000, 2000, 5000',
        items: 251
    },
    edge: {
        packSizes: '23, 31, 53',
        items: 500000
    },
    small: {
        packSizes: '10, 25, 50, 100',
        items: 137
    }
};

// Load preset configuration
function loadPreset(presetName) {
    const preset = presets[presetName];
    if (!preset) return;

    document.getElementById('pack_sizes').value = preset.packSizes;
    document.getElementById('items').value = preset.items;
}

// Format number with commas
function formatNumber(num) {
    return num.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ',');
}

// Validate form inputs
function validateForm() {
    const itemsInput = document.getElementById('items');
    const items = parseInt(itemsInput.value);

    if (isNaN(items) || items <= 0) {
        alert('Please enter a valid quantity greater than 0');
        return false;
    }

    const packSizesInput = document.getElementById('pack_sizes').value.trim();
    if (packSizesInput) {
        const packSizes = packSizesInput.split(',').map(s => parseInt(s.trim()));
        const invalidSizes = packSizes.filter(s => isNaN(s) || s <= 0);
        
        if (invalidSizes.length > 0) {
            alert('Please enter valid pack sizes (positive numbers separated by commas)');
            return false;
        }
    }

    return true;
}

// Initialize on page load
document.addEventListener('DOMContentLoaded', function() {
    console.log('Pack Calculator initialized');
    
    // Add form validation
    const form = document.getElementById('calc-form');
    if (form) {
        form.addEventListener('submit', function(e) {
            if (!validateForm()) {
                e.preventDefault();
                return false;
            }
        });
    }
});

