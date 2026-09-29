(() => {
  const form = document.getElementById('media-upload-form');
  const input = document.getElementById('media-file-input');
  const progress = document.getElementById('media-upload-progress');
  if (!form || !input || !progress) return;

  form.addEventListener('submit', event => {
    event.preventDefault();
    const files = [...input.files].slice(0, 50);
    if (!files.length) return;
    progress.replaceChildren();
    const queue = files.map(file => ({ file, retries: 0 }));
    let active = 0;
    let successes = 0;
    let failures = 0;
    const run = () => {
      while (active < 3 && queue.length) {
        const item = queue.shift();
        active++;
        upload(item).finally(() => {
          active--;
          // The XHR follows the 303 redirect, so a finished upload is already
          // committed server-side; reload so the new card appears without a
          // manual refresh. Failed uploads keep their inline retry buttons.
          if (!active && !queue.length && successes && !failures) {
            window.location.reload();
            return;
          }
          run();
        });
      }
    };
    run();

    function upload(item) {
      const row = document.createElement('div');
      const label = document.createElement('span');
      const bar = document.createElement('progress');
      const status = document.createElement('span');
      label.textContent = item.file.name;
      bar.max = 100;
      row.append(label, bar, status);
      progress.append(row);
      status.textContent = ' Yükleniyor';
      return new Promise(resolve => {
        const xhr = new XMLHttpRequest();
        xhr.open('POST', form.action);
        xhr.upload.onprogress = event => { if (event.lengthComputable) bar.value = Math.round(event.loaded / event.total * 100); };
        xhr.onload = () => {
          if (xhr.status >= 200 && xhr.status < 400) { successes++; bar.value = 100; status.textContent = ' Tamamlandı'; resolve(); return; }
          failed();
        };
        xhr.onerror = failed;
        const data = new FormData();
        data.append('file', item.file, item.file.name);
        data.append('visibility', form.elements.visibility.value);
        data.append('folder_id', form.elements.folder_id.value);
        data.append('_csrf', form.elements._csrf.value);
        xhr.send(data);
        function failed() {
          failures++;
          status.textContent = ' Yükleme başarısız';
          if (item.retries < 1) {
            const retry = document.createElement('button');
            retry.type = 'button'; retry.textContent = 'Yeniden dene';
            retry.addEventListener('click', () => { retry.remove(); item.retries++; failures--; upload(item); });
            row.append(retry);
          }
          resolve();
        }
      });
    }
  });

  document.addEventListener('click', event => {
    const opener = event.target.closest('[data-pdf-cover-open]');
    if (opener) {
      opener.closest('[data-pdf-cover-preview]')?.querySelector('dialog')?.showModal();
      return;
    }
    const close = event.target.closest('[data-pdf-cover-close]');
    if (close) {
      close.closest('dialog')?.close();
      return;
    }
    const preview = event.target.closest('.media-focal-preview');
    if (!preview) return;
    const card = preview.closest('.media-card');
    const editor = card && card.querySelector('form[action$="/metadata"]');
    if (!editor) return;
    const rect = preview.getBoundingClientRect();
    const x = Math.max(0, Math.min(1, (event.clientX - rect.left) / rect.width));
    const y = Math.max(0, Math.min(1, (event.clientY - rect.top) / rect.height));
    editor.elements.focal_x.value = x.toFixed(2);
    editor.elements.focal_y.value = y.toFixed(2);
    preview.style.objectPosition = `${(x * 100).toFixed(2)}% ${(y * 100).toFixed(2)}%`;
    editor.querySelector('summary').textContent = `Odak ${Math.round(x * 100)}%, ${Math.round(y * 100)}% · Klasör ve etiketler`;
  });

  document.querySelectorAll('.media-card').forEach(card => {
    card.draggable = true;
    card.addEventListener('dragstart', event => event.dataTransfer.setData('text/media-id', card.dataset.id || ''));
  });
  document.querySelectorAll('[data-media-folder-drop]').forEach(folder => {
    folder.addEventListener('dragover', event => event.preventDefault());
    folder.addEventListener('drop', event => {
      event.preventDefault();
      const card = [...document.querySelectorAll('.media-card')].find(item => item.dataset.id === event.dataTransfer.getData('text/media-id'));
      const editor = card && card.querySelector('.media-metadata-form');
      if (!editor) return;
      editor.elements.folder_id.value = folder.dataset.mediaFolderDrop;
      editor.requestSubmit();
    });
  });
})();
