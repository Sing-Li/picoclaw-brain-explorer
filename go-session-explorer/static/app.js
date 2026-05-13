let currentSessionKey = null;
let currentLogName = null;
let autoRefreshInterval = null;
let isEditing = false;
let isViewingLog = false;

function toggleAccordion(id) {
    const item = document.getElementById(id);
    const wasActive = item.classList.contains('active');
    
    // Close all
    document.querySelectorAll('.accordion-item').forEach(el => {
        el.classList.remove('active');
        el.querySelector('.accordion-icon').innerText = '▸';
    });

    // Toggle current
    if (!wasActive) {
        item.classList.add('active');
        item.querySelector('.accordion-icon').innerText = '▾';
    }
}

function filterList() {
    const input = document.getElementById('searchInput');
    const filter = input.value.toLowerCase();
    
    // Filter sessions
    const sessionList = document.getElementById('sessionList');
    const sessions = sessionList.getElementsByTagName('li');
    for (let i = 0; i < sessions.length; i++) {
        const key = sessions[i].getAttribute('data-key').toLowerCase();
        const summary = sessions[i].getAttribute('data-summary').toLowerCase();
        sessions[i].style.display = (key.indexOf(filter) > -1 || summary.indexOf(filter) > -1) ? "" : "none";
    }

    // Filter logs
    const logList = document.getElementById('logList');
    const logs = logList.getElementsByTagName('li');
    for (let i = 0; i < logs.length; i++) {
        const name = logs[i].getAttribute('data-name').toLowerCase();
        logs[i].style.display = (name.indexOf(filter) > -1) ? "" : "none";
    }

    // Filter cron jobs
    const cronList = document.getElementById('cronList');
    const crons = cronList.getElementsByTagName('li');
    for (let i = 0; i < crons.length; i++) {
        const name = crons[i].getAttribute('data-name').toLowerCase();
        crons[i].style.display = (name.indexOf(filter) > -1) ? "" : "none";
    }
}

async function loadSession(key) {
    if (isEditing && !confirm('You have unsaved changes. Refresh anyway?')) return;
    
    currentSessionKey = key;
    currentLogName = null;
    isEditing = false;
    isViewingLog = false;
    
    // Update active class across all lists
    document.querySelectorAll('.list-item').forEach(item => {
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
        
        // Default auto-refresh off for new load, or preserve if we wanted to
        stopAutoRefresh();
    } catch (error) {
        console.error('Failed to load session:', error);
    }
}

async function loadLog(name) {
    if (isEditing && !confirm('You have unsaved changes. Refresh anyway?')) return;

    currentLogName = name;
    currentSessionKey = null;
    isEditing = false;
    isViewingLog = true;
    stopAutoRefresh();

    // Update active class
    document.querySelectorAll('.list-item').forEach(item => {
        if (item.getAttribute('data-name') === name) {
            item.classList.add('active');
        } else {
            item.classList.remove('active');
        }
    });

    try {
        const response = await fetch(`/log?name=${encodeURIComponent(name)}`);
        const html = await response.text();
        document.getElementById('mainContent').innerHTML = html;
    } catch (error) {
        console.error('Failed to load log:', error);
    }
}

async function loadCron(id) {
    if (isEditing && !confirm('You have unsaved changes. Refresh anyway?')) return;

    currentSessionKey = null;
    currentLogName = null;
    isEditing = false;
    isViewingLog = false;
    stopAutoRefresh();

    // Update active class
    document.querySelectorAll('.list-item').forEach(item => {
        if (item.getAttribute('data-id') === id) {
            item.classList.add('active');
        } else {
            item.classList.remove('active');
        }
    });

    try {
        const response = await fetch(`/cron?id=${encodeURIComponent(id)}`);
        const html = await response.text();
        document.getElementById('mainContent').innerHTML = html;
    } catch (error) {
        console.error('Failed to load cron job:', error);
    }
}

function toggleCronEdit() {
    const view = document.getElementById('cronView');
    const edit = document.getElementById('cronEdit');
    const btn = document.getElementById('editCronBtn');
    
    if (view.style.display === 'none') {
        view.style.display = 'block';
        edit.style.display = 'none';
        btn.innerText = 'Edit Job';
        isEditing = false;
    } else {
        view.style.display = 'none';
        edit.style.display = 'block';
        btn.innerText = 'Cancel';
        isEditing = true;
    }
}

async function saveCronJob(id) {
    try {
        const name = document.getElementById('editCronName').value;
        const enabled = document.getElementById('editCronEnabled').checked;
        const scheduleKind = document.getElementById('editCronScheduleKind').value;
        const scheduleExpr = document.getElementById('editCronScheduleExpr').value;
        const payloadMessage = document.getElementById('editCronPayloadMessage').value;
        const payloadChannel = document.getElementById('editCronPayloadChannel').value;
        const payloadTo = document.getElementById('editCronPayloadTo').value;

        // Fetch original job to preserve fields not in form (like State, timestamps)
        // Note: For a real app, the API should handle partial updates, but we'll reconstruction here.
        // We'll use a hack: the list already has some data, but let's just send what we have.
        // The backend handler we wrote replaces the WHOLE job.
        
        // Let's get the full job first
        // We don't have a direct JSON API for full single job yet in handlers.go, 
        // wait, we have getCronJobs and we can add a simple JSON getter if needed.
        // Actually, let's just fetch all cron jobs as JSON from /api/sessions? No, that's for sessions.
        // I'll add /api/cron to handlers.go to list all jobs as JSON.
        
        const res = await fetch(`/api/cron/${encodeURIComponent(id)}`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                id: id,
                name: name,
                enabled: enabled,
                schedule: { kind: scheduleKind, expr: scheduleExpr },
                payload: { kind: 'agent_turn', message: payloadMessage, channel: payloadChannel, to: payloadTo }
            })
        });

        if (res.ok) {
            isEditing = false;
            window.location.reload(); // Reload to update list and view
        } else {
            alert('Failed to save job');
        }
    } catch (error) {
        console.error('Failed to save cron job:', error);
    }
}

async function deleteCronJob(id) {
    if (!confirm('Are you sure you want to delete this cron job?')) return;
    
    try {
        const response = await fetch(`/api/cron/${encodeURIComponent(id)}`, {
            method: 'DELETE'
        });
        if (response.ok) {
            window.location.reload();
        } else {
            alert('Failed to delete job');
        }
    } catch (error) {
        console.error('Failed to delete cron job:', error);
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
