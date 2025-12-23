document.addEventListener('DOMContentLoaded', function () {
  const uploadArea = document.getElementById('uploadArea');
  const fileInput = document.getElementById('fileInput');
  const uploadForm = document.getElementById('uploadForm') || document.getElementById('imageUploadForm');

  if (uploadArea && fileInput && uploadForm) {
    uploadArea.addEventListener('click', function () { fileInput.click() });
    fileInput.addEventListener('change', function () {
      if (fileInput.files.length > 0) {
        handleUpload(fileInput.files, uploadForm);
      }
    });
    uploadArea.addEventListener('dragover', function (e) { e.preventDefault(); uploadArea.classList.add('dragover') });
    uploadArea.addEventListener('dragleave', function (e) { e.preventDefault(); uploadArea.classList.remove('dragover') });
    uploadArea.addEventListener('drop', function (e) {
      e.preventDefault(); uploadArea.classList.remove('dragover');
      const files = e.dataTransfer.files;
      if (files.length > 0) {
        fileInput.files = files;
        handleUpload(files, uploadForm);
      }
    });
  }
});

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
  const uploadPromises = [];

  for (let i = 0; i < files.length; i++) {
    const file = files[i];
    const formData = new FormData();
    formData.append('image', file);
    formData.append('album_id', albumID);

    uploadPromises.push(
      fetch('/upload', {
        method: 'POST',
        body: formData,
        credentials: 'same-origin'
      }).then(response => {
        if (!response.ok) {
          throw new Error('Upload failed for ' + file.name);
        }
        return response;
      })
    );
  }

  Promise.all(uploadPromises)
    .then(() => {
      // Перенаправляем в альбом
      window.location.href = '/' + sessionID + '/' + albumID;
    })
    .catch(error => {
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
    <div class="empty-icon">📷</div>
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
        imageItem.style.opacity = '0.5';
        setTimeout(() => {
          imageItem.remove();
          // Проверяем, остались ли изображения
          const remainingImages = document.querySelectorAll('.image-item');
          if (remainingImages.length === 0) {
            // Показываем пустое состояние
            const imageGrid = document.getElementById('imageGrid');
            imageGrid.innerHTML = EMPTY_STATE_HTML;
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
