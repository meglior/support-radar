use std::time::Duration;
use tokio::time::sleep;
use serde::Serialize;
use sysinfo::{System, SystemExt, DiskExt};

// Структура Heartbeat, которая ОДИН В ОДИН совпадает с тем, что ждет наш Go-сервер
#[derive(Serialize)]
struct Heartbeat {
    agent_version: String,
    integrity_hash: String,
    timestamp: i64,
    machine_name: String,
    current_user: String,
    last_boot: i64,
    free_disk_space_gb: f64,
    status: String,
}

#[tokio::main]
async fn main() {
    println!("Запуск агента Support-Radar под Windows...");

    let mut sys = System::new_all();

    // Бесконечный цикл отправки "пульса"
    loop {
        // Обновляем системные данные
        sys.refresh_all();

        // Собираем имя машины
        let machine_name = sys.host_name().unwrap_or_else(|| "Unknown-PC".to_string());
        
        // Считаем свободное место на диске C:\
        let mut free_space = 0.0;
        for disk in sys.disks() {
            if disk.mount_point().to_str() == Some("C:\\") || disk.mount_point().to_str() == Some("/") {
                free_space = disk.available_space() as f64 / 1024.0 / 1024.0 / 1024.0; // В Гигабайты
            }
        }

        // Формируем структуру хартбита
        let hb = Heartbeat {
            agent_version: "1.0.0".to_string(),
            integrity_hash: "system_clean_hash_v1".to_string(),
            timestamp: chrono::Utc::now().timestamp(), // Для этого позже добавим chrono, пока можно заглушку или убрать
            machine_name,
            current_user: "SYSTEM".to_string(), // В будущем стянем через winapi
            last_boot: sys.uptime() as i64,
            free_disk_space_gb: (free_space * 100.0).round() / 100.0,
            status: "online".to_string(),
        };

        // Сериализуем в JSON строчку
        if let Ok(json_str) = serde_json::to_string(&hb) {
            println!("Сформирован хартбит: {}", json_str);
            // TODO: Отправка в вебсокет с mTLS сертификатом
        }

        // Ждем 5 секунд перед следующим пингом
        sleep(Duration::from_secs(5)).await;
    }
}