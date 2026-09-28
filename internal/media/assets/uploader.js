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
    const run = () => {
      while (active < 3 && queue.length) {
        const item = queue.shift();
        active++;
        upload(item).finally(() => { active--; run(); });
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
          if (xhr.status >= 200 && xhr.status < 400) { bar.value = 100; status.textContent = ' Tamamlandı'; resolve(); return; }
          failed();
        };
        xhr.onerror = failed;
        const data = new FormData();
        data.append('file', item.file, item.file.name);
        data.append('visibility', form.elements.visibility.value);
        data.append('_csrf', form.elements._csrf.value);
        xhr.send(data);
        function failed() {
          status.textContent = ' Yükleme başarısız';
          if (item.retries < 1) {
            const retry = document.createElement('button');
            retry.type = 'button'; retry.textContent = 'Yeniden dene';
            retry.addEventListener('click', () => { retry.remove(); item.retries++; upload(item); });
            row.append(retry);
          }
          resolve();
        }
      });
    }
  });
})();
