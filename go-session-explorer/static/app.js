let currentSessionKey = null;
let autoRefreshInterval = null;
let isEditing = false;

function filterSessions() {
    const input = document.getElementById('searchInput');
    const filter = input.value.toLowerCase();
    const ul = document.getElementById('sessionList');
    const li = ul.getElementsByTagName('li');

    for (let i = 0; i < li.length; i++) {
        const key = li[i].getAttribute('data-key').toLowerCase();
        const summary = li[i].getAttribute('data-summary').toLowerCase();
        if (key.indexOf(filter) > -1 || summary.indexOf(filter) > -1) {
            li[i].style.display = "";
        } else {
            li[i].style.display = "none";
        }
    }
}

async function loadSession(key) {
    if (isEditing && !confirm('You have unsaved changes. Refresh anyway?')) return;
    
    currentSessionKey = key;
    isEditing = false;
    
    // Preserve auto-refresh state if element exists
    const autoRefreshWasOn = document.getElementById('autoRefreshCheck')?.checked;

    // Update active class
    const items = document.querySelectorAll('.session-item');
    items.forEach(item => {
        if (item.getAttribute('data-key') === key) {
            item.classList.add('active');
        } else {
            item.classList.remove('active');
        }
    });

    try {
        const response = await fetch(`/session?key=${encodeURIComponent(key)}`);
        const html = await response.text();
        document.getElementById('mainContent').innerHTML = html;
        
        // Restore auto-refresh state
        if (autoRefreshWasOn) {
            document.getElementById('autoRefreshCheck').checked = true;
            startAutoRefresh();
        }
    } catch (error) {
        console.error('Failed to load session:', error);
    }
}

function toggleAutoRefresh(enabled) {
    if (enabled) {
        startAutoRefresh();
    } else {
        stopAutoRefresh();
    }
}

function startAutoRefresh() {
    stopAutoRefresh();
    autoRefreshInterval = setInterval(() => {
        if (currentSessionKey && !isEditing) {
            refreshSessionSilently(currentSessionKey);
        }
    }, 3000); // Refresh every 3 seconds
}

function stopAutoRefresh() {
    if (autoRefreshInterval) {
        clearInterval(autoRefreshInterval);
        autoRefreshInterval = null;
    }
}

async function refreshSessionSilently(key) {
    try {
        const response = await fetch(`/session?key=${encodeURIComponent(key)}`);
        const html = await response.text();
        
        // Only update if not editing and session is still the same
        if (currentSessionKey === key && !isEditing) {
            const container = document.getElementById('mainContent');
            // We use a temporary div to compare content to avoid flickering if nothing changed
            const temp = document.createElement('div');
            temp.innerHTML = html;
            
            // Simple comparison of message lists to reduce flicker
            const currentList = container.querySelector('.message-list');
            const newList = temp.querySelector('.message-list');
            
            if (newList && (!currentList || currentList.innerHTML !== newList.innerHTML)) {
                // Preserve header state (like checkbox)
                const autoRefreshCheck = document.getElementById('autoRefreshCheck');
                const wasChecked = autoRefreshCheck ? autoRefreshCheck.checked : false;
                
                container.innerHTML = html;
                
                if (wasChecked) {
                    document.getElementById('autoRefreshCheck').checked = true;
                }
            }
        }
    } catch (error) {
        console.error('Failed silent refresh:', error);
    }
}

async function deleteSession(key) {
    if (!confirm('Are you sure you want to delete this session?')) return;
    
    try {
        const response = await fetch(`/api/sessions/${encodeURIComponent(key)}`, {
            method: 'DELETE'
        });
        if (response.ok) {
            window.location.reload(); // Reload the whole page to update the list
        }
    } catch (error) {
        console.error('Failed to delete session:', error);
    }
}

function editMessage(index) {
    isEditing = true;
    document.getElementById(`content-${index}`).style.display = 'none';
    document.getElementById(`edit-${index}`).style.display = 'block';
}

function cancelEdit(index) {
    isEditing = false;
    document.getElementById(`content-${index}`).style.display = 'block';
    document.getElementById(`edit-${index}`).style.display = 'none';
}

async function saveMessage(index) {
    const newContent = document.getElementById(`textarea-${index}`).value;
    const summary = document.getElementById('currentSessionSummary').innerText;
    
    // We need to fetch the whole session, update the message, and PUT it back
    // Since we rendered HTML, we'll fetch the JSON data first to reconstruct it.
    try {
        const getRes = await fetch(`/api/sessions/${encodeURIComponent(currentSessionKey)}`);
        const sessionData = await getRes.json();
        
        sessionData.messages[index].content = newContent;
        sessionData.meta.summary = summary === 'No summary' ? '' : summary;

        const putRes = await fetch(`/api/sessions/${encodeURIComponent(currentSessionKey)}`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(sessionData)
        });

        if (putRes.ok) {
            isEditing = false;
            // Reload the session HTML
            loadSession(currentSessionKey);
        }
    } catch (error) {
        console.error('Failed to save message:', error);
    }
}

async function deleteMessage(index) {
    if (!confirm('Are you sure you want to delete this message?')) return;

    try {
        const getRes = await fetch(`/api/sessions/${encodeURIComponent(currentSessionKey)}`);
        const sessionData = await getRes.json();
        
        sessionData.messages.splice(index, 1);
        const summary = document.getElementById('currentSessionSummary').innerText;
        sessionData.meta.summary = summary === 'No summary' ? '' : summary;

        const putRes = await fetch(`/api/sessions/${encodeURIComponent(currentSessionKey)}`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(sessionData)
        });

        if (putRes.ok) {
            isEditing = false;
            loadSession(currentSessionKey);
        }
    } catch (error) {
        console.error('Failed to delete message:', error);
    }
}
