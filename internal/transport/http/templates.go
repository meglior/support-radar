package http

const indexTemplate = `<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Support-Radar — Панель управления</title>
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.2/dist/css/bootstrap.min.css" rel="stylesheet">
    <style>
        .status-online { color: #28a745; }
        .status-offline { color: #dc3545; }
        .status-maintenance { color: #ffc107; }
        .status-fallback { color: #fd7e14; }
    </style>
</head>
<body class="bg-light">
    <nav class="navbar navbar-dark bg-dark">
        <div class="container">
            <span class="navbar-brand">🎯 Support-Radar</span>
            <span class="text-light">Агентов онлайн: <strong>{{.OnlineCount}}</strong></span>
        </div>
    </nav>
    
    <div class="container mt-4">
        <div class="row">
            <div class="col-12">
                <div class="card">
                    <div class="card-header d-flex justify-content-between align-items-center">
                        <h5 class="mb-0">Хосты</h5>
                        <form class="d-flex" method="GET" action="/">
                            <input class="form-control me-2" type="search" name="search" 
                                   placeholder="Имя ПК, IP или пользователь" value="{{.Search}}">
                            <button class="btn btn-outline-primary" type="submit">Поиск</button>
                        </form>
                    </div>
                    <div class="card-body">
                        <table class="table table-hover">
                            <thead>
                                <tr>
                                    <th>Статус</th>
                                    <th>Имя ПК</th>
                                    <th>IP</th>
                                    <th>Пользователь</th>
                                    <th>Версия</th>
                                    <th>Последний heartbeat</th>
                                    <th>Действия</th>
                                </tr>
                            </thead>
                            <tbody>
                                {{range .Endpoints}}
                                <tr>
                                    <td>
                                        {{if eq .Status "ONLINE"}}
                                            <span class="badge bg-success">ONLINE</span>
                                        {{else if eq .Status "MAINTENANCE"}}
                                            <span class="badge bg-warning">MAINTENANCE</span>
                                        {{else if eq .Status "FALLBACK"}}
                                            <span class="badge bg-warning">FALLBACK</span>
                                        {{else}}
                                            <span class="badge bg-secondary">OFFLINE</span>
                                        {{end}}
                                    </td>
                                    <td><strong>{{.MachineName}}</strong></td>
                                    <td>{{.ActiveIP}}</td>
                                    <td>{{.DomainName}}</td>
                                    <td><code>{{.AgentVersion}}</code></td>
                                    <td>{{.LastHeartbeat.Format "02.01 15:04"}}</td>
                                    <td>
                                        <a href="/host/{{.ID}}" class="btn btn-sm btn-primary">Управление</a>
                                    </td>
                                </tr>
                                {{else}}
                                <tr>
                                    <td colspan="7" class="text-center text-muted">Хосты не найдены</td>
                                </tr>
                                {{end}}
                            </tbody>
                        </table>
                    </div>
                </div>
            </div>
        </div>
    </div>
    
    <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.3.2/dist/js/bootstrap.bundle.min.js"></script>
</body>
</html>`

const hostTemplate = `<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Support-Radar — {{.Endpoint.MachineName}}</title>
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.2/dist/css/bootstrap.min.css" rel="stylesheet">
    <style>
        .status-online { color: #28a745; }
        .status-offline { color: #dc3545; }
        .command-btn { margin-bottom: 0.5rem; }
    </style>
</head>
<body class="bg-light">
    <nav class="navbar navbar-dark bg-dark">
        <div class="container">
            <a href="/" class="navbar-brand">🎯 Support-Radar</a>
            <span class="text-light">{{.Endpoint.MachineName}}</span>
        </div>
    </nav>
    
    <div class="container mt-4">
        <div class="row">
            <div class="col-md-4">
                <div class="card">
                    <div class="card-header">
                        <h5 class="mb-0">Информация</h5>
                    </div>
                    <div class="card-body">
                        <table class="table table-sm">
                            <tr><td><strong>Статус:</strong></td>
                                <td>
                                    {{if .IsOnline}}
                                        <span class="badge bg-success">🟢 Онлайн</span>
                                    {{else}}
                                        <span class="badge bg-danger">🔴 Оффлайн</span>
                                    {{end}}
                                </td>
                            </tr>
                            <tr><td><strong>Имя ПК:</strong></td><td>{{.Endpoint.MachineName}}</td></tr>
                            <tr><td><strong>IP:</strong></td><td>{{.Endpoint.ActiveIP}}</td></tr>
                            <tr><td><strong>Домен:</strong></td><td>{{.Endpoint.DomainName}}</td></tr>
                            <tr><td><strong>Версия агента:</strong></td><td><code>{{.Endpoint.AgentVersion}}</code></td></tr>
                            <tr><td><strong>Hash:</strong></td><td><small class="text-muted">{{.Endpoint.IntegrityHash}}</small></td></tr>
                            <tr><td><strong>Последний heartbeat:</strong></td><td>{{.Endpoint.LastHeartbeat.Format "02.01.2006 15:04:05"}}</td></tr>
                        </table>
                    </div>
                </div>
                
                <div class="card mt-3">
                    <div class="card-header">
                        <h5 class="mb-0">Команды</h5>
                    </div>
                    <div class="card-body">
                        {{if .IsOnline}}
                            {{range .Commands}}
                            <button class="btn btn-outline-primary btn-sm w-100 command-btn" 
                                    onclick="executeCommand('{{.}}')">
                                {{.}}
                            </button>
                            {{end}}
                        {{else}}
                            <div class="alert alert-warning">Агент оффлайн. Команды недоступны.</div>
                        {{end}}
                    </div>
                </div>
            </div>
            
            <div class="col-md-8">
                <div class="card">
                    <div class="card-header">
                        <h5 class="mb-0">История действий</h5>
                    </div>
                    <div class="card-body">
                        <table class="table table-sm">
                            <thead>
                                <tr>
                                    <th>Время</th>
                                    <th>Инженер</th>
                                    <th>Команда</th>
                                    <th>Статус</th>
                                </tr>
                            </thead>
                            <tbody>
                                {{range .Logs}}
                                <tr>
                                    <td>{{.Timestamp.Format "02.01 15:04"}}</td>
                                    <td>{{.EngineerUsername}} <span class="badge bg-secondary">{{.EngineerRole}}</span></td>
                                    <td><code>{{.CommandID}}</code></td>
                                    <td>
                                        {{if eq .ExecutionStatus "SUCCESS"}}
                                            <span class="badge bg-success">✓</span>
                                        {{else if eq .ExecutionStatus "PENDING"}}
                                            <span class="badge bg-warning">⏳</span>
                                        {{else}}
                                            <span class="badge bg-danger">✗</span>
                                        {{end}}
                                    </td>
                                </tr>
                                {{else}}
                                <tr><td colspan="4" class="text-center text-muted">История пуста</td></tr>
                                {{end}}
                            </tbody>
                        </table>
                    </div>
                </div>
            </div>
        </div>
    </div>
    
    <script>
    function executeCommand(cmdId) {
        if (!confirm('Выполнить команду ' + cmdId + '?')) return;
        
        fetch('/api/v1/endpoints/{{.Endpoint.ID}}/execute', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                transaction_id: 'tx_' + Date.now(),
                commands: [{ command_id: cmdId, args: {} }]
            })
        })
        .then(r => {
            if (r.status === 202) {
                alert('Команда отправлена агенту');
                location.reload();
            } else if (r.status === 429) {
                alert('Rate limit: слишком много команд. Подождите 5 минут.');
            } else {
                alert('Ошибка: ' + r.status);
            }
        })
        .catch(e => alert('Ошибка сети: ' + e));
    }
    </script>
    <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.3.2/dist/js/bootstrap.bundle.min.js"></script>
</body>
</html>`
