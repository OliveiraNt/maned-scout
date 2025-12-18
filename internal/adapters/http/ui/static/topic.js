function showUpdateConfigModal() {
    document.getElementById('updateConfigModal').classList.remove('hidden');
}

function closeUpdateConfigModal() {
    document.getElementById('updateConfigModal').classList.add('hidden');
}

function showIncreasePartitionsModal() {
    document.getElementById('increasePartitionsModal').classList.remove('hidden');
}

function closeIncreasePartitionsModal() {
    document.getElementById('increasePartitionsModal').classList.add('hidden');
}

function confirmDeleteTopic() {
    document.getElementById('deleteTopicModal').classList.remove('hidden');
}

function closeDeleteTopicModal() {
    document.getElementById('deleteTopicModal').classList.add('hidden');
}

function showWriteMessageModal() {
    document.getElementById('writeMessageModal').classList.remove('hidden');
}
function closeWriteMessageModal() {
    document.getElementById('writeMessageModal').classList.add('hidden');
}

async function updateTopicConfig(event) {
    event.preventDefault();
    const form = event.target;
    const formData = new FormData(form);

    const configs = {};
    for (const [key, value] of formData.entries()) {
        if (value) {
            configs[key] = value;
        }
    }

    if (Object.keys(configs).length === 0) {
        showNotification('Nenhuma configuração foi modificada', 'error');
        return;
    }

    try {
        const response = await fetch(`/api/clusters/${clusterName}/topics/${topicName}/config`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ configs })
        });

        if (response.ok) {
            queueNotification('Configurações atualizadas com sucesso!', 'success');
            location.reload();
        } else {
            const error = await response.text();
            showNotification(`Erro: ${error}`, 'error');
        }
    } catch (error) {
        showNotification(`Erro: ${error.message}`, 'error');
    }
}

async function increasePartitions(event) {
    event.preventDefault();
    const form = event.target;
    const formData = new FormData(form);
    const totalPartitions = parseInt(formData.get('totalPartitions'));

    try {
        const response = await fetch(`/api/clusters/${clusterName}/topics/${topicName}/partitions`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ totalPartitions })
        });

        if (response.ok) {
            showNotification('Partições aumentadas com sucesso!', 'success');
            closeIncreasePartitionsModal();
            location.reload();
        } else {
            const error = await response.text();
            showNotification(`Erro: ${error}`, 'error');
        }
    } catch (error) {
        showNotification(`Erro: ${error.message}`, 'error');
    }
}

async function deleteTopic() {
    try {
        const response = await fetch(`/api/clusters/${clusterName}/topics/${topicName}`, {
            method: 'DELETE'
        });

        if (response.ok) {
            queueNotification('Tópico deletado com sucesso!', 'success');
            window.location.href = `/clusters/${clusterName}/topics`;
        } else {
            const error = await response.text();
            showNotification(`Erro: ${error}`, 'error');
        }
    } catch (error) {
        showNotification(`Erro: ${error.message}`, 'error');
    }
}

async function sendMessage(event) {
    event.preventDefault();
    const form = event.target;
    const formData = new FormData(form);
    const key = formData.get('key');
    const value = formData.get('value');
    
    try {
        const response = await fetch(`/api/clusters/${clusterName}/topics/${topicName}/messages`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ key, value })
        });
        
        if (response.ok) {
            showNotification('Mensagem enviada com sucesso!', 'success');
            closeWriteMessageModal();
        } else {
            const error = await response.text();
            showNotification(`Erro: ${error}`, 'error');
        }
    } catch (error) {
        showNotification(`Erro: ${error.message}`, 'error');
    }
}
