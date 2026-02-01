document.addEventListener('DOMContentLoaded', function () {
  const uploadArea = document.getElementById('uploadArea');
  const fileInput = document.getElementById('fileInput');
  const uploadForm = document.getElementById('uploadForm') || document.getElementById('imageUploadForm');

  if (uploadArea && fileInput && uploadForm) {
    uploadArea.addEventListener('click', function () { fileInput.click() }, { passive: true });
    fileInput.addEventListener('change', function () {
      if (fileInput.files.length > 0) {
        handleUpload(fileInput.files, uploadForm);
      }
    });
    uploadArea.addEventListener('dragover', function (e) {
      e.preventDefault();
      uploadArea.classList.add('dragover');
    });
    uploadArea.addEventListener('dragleave', function (e) {
      e.preventDefault();
      uploadArea.classList.remove('dragover');
    });
    uploadArea.addEventListener('drop', function (e) {
      e.preventDefault();
      uploadArea.classList.remove('dragover');
      const files = e.dataTransfer.files;
      if (files.length > 0) {
        fileInput.files = files;
        handleUpload(files, uploadForm);
      }
    });
  }

  // Инициализация темы
  initTheme();

  // Инициализация иконок Lucide
  if (window.lucide) {
    lucide.createIcons();
  }
});

// Функции для работы с темами
function initTheme() {
  const savedTheme = localStorage.getItem('ripx_theme') || 'crystal';
  applyTheme(savedTheme);

  const themeSelect = document.getElementById('themeSelect');
  if (themeSelect) {
    themeSelect.value = savedTheme;
  }
}

function changeTheme(themeName) {
  applyTheme(themeName);
  localStorage.setItem('ripx_theme', themeName);
}

function applyTheme(themeName) {
  if (themeName === 'crystal') {
    document.documentElement.removeAttribute('data-theme');
  } else {
    document.documentElement.setAttribute('data-theme', themeName);
  }
  // Перерисовываем иконки, если нужно (например, если они зависят от темы)
  if (window.lucide) {
    lucide.createIcons();
  }
}

// Показать индикатор загрузки
function showUploadProgress(total) {
  const overlay = document.getElementById('uploadOverlay');
  const status = document.getElementById('uploadStatus');
  const count = document.getElementById('uploadCount');

  overlay.classList.add('active');
  status.textContent = 'зᴀᴦᴩузᴋᴀ...';
  count.textContent = '0 / ' + total + ' файлов';

  return {
    update: function (current) {
      status.textContent = 'зᴀᴦᴩузᴋᴀ...';
      count.textContent = current + ' / ' + total + ' файлов';
    },
    hide: function () {
      overlay.classList.remove('active');
    }
  };
}

// handleUpload обрабатывает загрузку файлов
function handleUpload(files, form) {
  const albumInput = form.querySelector('input[name="album_id"]');

  // Если album_id уже есть в форме (загрузка в существующий альбом)
  if (albumInput && albumInput.value) {
    // sessionID из URL текущей страницы
    const pathParts = window.location.pathname.split('/').filter(p => p);
    const sessionID = pathParts[0] || '';
    uploadFilesParallel(files, albumInput.value, sessionID);
    return;
  }

  // Иначе создаем новый альбом на сервере
  fetch('/create-album', {
    method: 'POST',
    credentials: 'same-origin'
  })
    .then(response => response.json())
    .then(data => {
      if (data.album_id && data.session_id) {
        uploadFilesParallel(files, data.album_id, data.session_id);
      } else {
        throw new Error('Failed to create album');
      }
    })
    .catch(error => {
      console.error('Error creating album:', error);
      alert('Ошибка при создании альбома');
    });
}

// uploadFilesParallel отправляет файлы параллельно
function uploadFilesParallel(files, albumID, sessionID) {
  const total = files.length;
  let completed = 0;
  const progress = showUploadProgress(total);

  // Создаем промисы для каждой операции конвертации и загрузки
  const uploadPromises = [];

  for (let i = 0; i < files.length; i++) {
    // Для каждого файла создаем цепочку: конвертация -> загрузка
    const uploadPromise = convertToWebP(files[i])
      .then(convertedFile => ({ file: convertedFile, originalFile: files[i] }))
      .catch(error => {
        console.error('Error converting image to WebP:', error);
        // Если конвертация не удалась, возвращаем оригинальный файл
        return { file: files[i], originalFile: files[i] };
      })
      .then(({ file, originalFile }) => {
        const formData = new FormData();
        formData.append('image', file);
        formData.append('album_id', albumID);

        return fetch('/upload', {
          method: 'POST',
          body: formData,
          credentials: 'same-origin',
          headers: {
            'X-Requested-With': 'XMLHttpRequest'
          }
        }).then(response => {
          if (!response.ok) {
            throw new Error('Upload failed for ' + file.name);
          }
          completed++;
          progress.update(completed);
          return response;
        });
      });

    uploadPromises.push(uploadPromise);
  }

  // Ждем завершения всех операций загрузки
  Promise.all(uploadPromises)
    .then(() => {
      progress.hide();
      // Перенаправляем в альбом
      window.location.href = '/' + sessionID + '/' + albumID;
    })
    .catch(error => {
      progress.hide();
      console.error('Upload error:', error);
      alert('Ошибка при загрузке: ' + error.message);
    });
}

// getSessionID получает ID сессии из cookie
function getSessionID() {
  const cookies = document.cookie.split(';');
  for (let i = 0; i < cookies.length; i++) {
    const cookie = cookies[i].trim();
    if (cookie.indexOf('session_id=') === 0) {
      return cookie.substring('session_id='.length, cookie.length);
    }
  }
  return '';
}

// HTML шаблон для пустого состояния (используется в deleteImage и album.html)
const EMPTY_STATE_HTML = `
  <div class="empty-state">
    <div class="empty-icon"><i data-lucide="image-off"></i></div>
    <div class="empty-text">у ʙᴀᴄ ᴨоᴋᴀ нᴇᴛ зᴀᴦᴩужᴇнных изобᴩᴀжᴇний</div>
    <a href="/" class="empty-link">зᴀᴦᴩузиᴛь ᴨᴇᴩʙоᴇ изобᴩᴀжᴇниᴇ</a>
  </div>
`;

// showCopiedFeedback показывает визуальную обратную связь о копировании
function showCopiedFeedback(button) {
  const originalText = button.textContent;
  button.textContent = 'ᴄᴋоᴨиᴩоʙᴀно!';
  button.classList.add('copied');
  setTimeout(function () {
    button.textContent = originalText;
    button.classList.remove('copied');
  }, 2000);
}

// Функция для копирования ссылки на альбом
function copyAlbumUrl(sessionID, albumID, button) {
  const url = window.location.origin + '/' + sessionID + '/' + albumID;
  if (navigator.clipboard) {
    navigator.clipboard.writeText(url)
      .then(function () { showCopiedFeedback(button) })
      .catch(function (err) { console.error('нᴇ удᴀᴧоᴄь ᴄᴋоᴨиᴩоʙᴀᴛь ᴜʀʟ: ', err) });
  } else {
    // Fallback для старых браузеров
    const textArea = document.createElement('textarea');
    textArea.value = url;
    document.body.appendChild(textArea);
    textArea.focus();
    textArea.select();
    try {
      document.execCommand('copy');
      showCopiedFeedback(button);
    } catch (err) { console.error('Не удалось скопировать URL: ', err) }
    document.body.removeChild(textArea);
  }
}

function copyUrl(sessionID, albumID, filename, button) {
  const url = window.location.origin + '/' + sessionID + '/' + albumID + '/' + filename;
  if (navigator.clipboard) {
    navigator.clipboard.writeText(url)
      .then(function () { showCopiedFeedback(button) })
      .catch(function (err) { console.error('нᴇ удᴀᴧоᴄь ᴄᴋоᴨиᴩоʙᴀᴛь ᴜʀʟ: ', err) });
  } else {
    // Fallback для старых браузеров
    const textArea = document.createElement('textarea');
    textArea.value = url;
    document.body.appendChild(textArea);
    textArea.focus();
    textArea.select();
    try {
      document.execCommand('copy');
      showCopiedFeedback(button);
    } catch (err) { console.error('Не удалось скопировать URL: ', err) }
    document.body.removeChild(textArea);
  }
}

function deleteImage(sessionID, albumID, filename, button) {
  if (!confirm('Вы уверены, что хотите удалить это изображение?')) {
    return;
  }

  const formData = new FormData();
  formData.append('album_id', albumID);
  formData.append('filename', filename);

  fetch('/delete-image', {
    method: 'POST',
    body: formData
  })
    .then(response => {
      if (response.ok) {
        // Удаляем элемент изображения из DOM
        const imageItem = button.closest('.image-item');
        imageItem.style.transition = 'opacity 0.3s ease';
        imageItem.style.opacity = '0';
        setTimeout(() => {
          imageItem.remove();
          // Проверяем, остались ли изображения
          const imageGrid = document.getElementById('imageGrid');
          if (!imageGrid.querySelector('.image-item')) {
            // Показываем пустое состояние
            imageGrid.innerHTML = EMPTY_STATE_HTML;
            if (window.lucide) {
              lucide.createIcons();
            }
          }
        }, 300);
      } else {
        alert('Ошибка при удалении изображения');
      }
    })
    .catch(error => {
      console.error('Error:', error);
      alert('Ошибка при удалении изображения');
    });
}

function deleteUser() {
  if (!confirm('Вы уверены, что хотите удалить весь профиль со всеми альбомами и изображениями? Это действие необратимо!')) {
    return;
  }

  fetch('/delete-user', {
    method: 'POST'
  })
    .then(response => {
      if (response.ok) {
        // Перезагружаем страницу - сервер уже очистил cookie
        window.location.href = '/';
      } else {
        alert('Ошибка при удалении профиля');
      }
    })
    .catch(error => {
      console.error('Error:', error);
      alert('Ошибка при удалении профиля');
    });
}

// Открывает изображение в оверлее
function toggleZoom(img) {
  const overlay = document.getElementById('image-viewer-overlay');
  const zoomedImageContainer = document.getElementById('zoomed-image-element');

  // Быстрая очистка контейнера
  zoomedImageContainer.textContent = '';

  // Создаем новое изображение вместо клонирования
  const newImg = document.createElement('img');
  newImg.src = img.src;
  newImg.alt = img.alt;
  newImg.loading = 'eager'; // Приоритетная загрузка для зума

  zoomedImageContainer.appendChild(newImg);
  overlay.classList.add('active');
}

// Закрывает оверлей
function closeZoom() {
  const overlay = document.getElementById('image-viewer-overlay');
  overlay.classList.remove('active');
}

// convertToWebP конвертирует изображение в формат WebP
function convertToWebP(file) {
  return new Promise((resolve, reject) => {
    // Проверяем, является ли файл изображением
    if (!file.type.startsWith('image/')) {
      reject(new Error('File is not an image'));
      return;
    }

    // Создаем объект FileReader для чтения файла
    const reader = new FileReader();
    reader.onload = function (e) {
      // Создаем элемент img для загрузки изображения
      const img = new Image();
      img.onload = function () {
        // Создаем canvas элемент для конвертации
        const canvas = document.createElement('canvas');
        canvas.width = img.width;
        canvas.height = img.height;

        const ctx = canvas.getContext('2d');
        // Рисуем изображение на canvas
        ctx.drawImage(img, 0, 0);

        // Конвертируем canvas в WebP формат
        canvas.toBlob(function (blob) {
          if (blob) {
            // Создаем новый File объект с правильным именем и типом
            const fileName = file.name.replace(/\.[^/.]+$/, '') + '.webp';
            const webpFile = new File([blob], fileName, { type: 'image/webp' });
            resolve(webpFile);
          } else {
            reject(new Error('Failed to convert image to WebP'));
          }
        }, 'image/webp', 0.85); // Качество 85%
      };
      img.onerror = function () {
        reject(new Error('Failed to load image'));
      };
      img.src = e.target.result;
    };
    reader.onerror = function () {
      reject(new Error('Failed to read file'));
    };
    reader.readAsDataURL(file);
  });
}

// Работа с ченджлогом
function checkChangelog() {
  fetch('/changelog')
    .then(response => response.json())
    .then(data => {
      if (!data.data || !data.data.content) return;

      const content = data.data.content;
      // Находим последнюю версию (первый заголовок ##)
      const versionMatch = content.match(/## \[?([\d.]+)\]?/);
      if (!versionMatch) return;

      const latestVersion = versionMatch[1];
      const savedVersion = localStorage.getItem('last_seen_version');

      if (latestVersion !== savedVersion) {
        showChangelog(content, latestVersion);
      }
    })
    .catch(error => console.error('Error fetching changelog:', error));
}

function showChangelog(content, version) {
  const modal = document.getElementById('changelogModal');
  const body = document.getElementById('changelogBody');

  if (!modal || !body) return;

  // Берем только последнюю секцию до следующего ## или конца файла
  const parts = content.split(/## \[?[\d.]+\]?/);
  // parts[0] - заголовок # Changelog и все что до первой версии
  // parts[1] - контент последней версии
  let latestContent = parts[1] || "";

  // Простой парсинг Markdown (заголовки и списки)
  let html = latestContent
    .trim()
    .replace(/^### (.*$)/gim, '<h3>$1</h3>')
    .replace(/^\- (.*$)/gim, '<li>$1</li>');

  // Группируем li в ul
  html = html.replace(/(<li>.*<\/li>(\n<li>.*<\/li>)*)/g, '<ul>$1</ul>');

  body.innerHTML = html;
  modal.dataset.version = version;
  modal.classList.add('active');
  document.body.style.overflow = 'hidden'; // Запрещаем прокрутку фона
}

function closeChangelog() {
  const modal = document.getElementById('changelogModal');
  const version = modal.dataset.version;

  if (version) {
    localStorage.setItem('last_seen_version', version);
  }

  modal.classList.remove('active');
  document.body.style.overflow = '';
}

// Инициализация при загрузке
document.addEventListener('DOMContentLoaded', function () {
  // Вызываем проверку ченджлога через небольшую задержку для плавности
  setTimeout(checkChangelog, 1000);
});


