(() => {
  const SLOT_COUNT = 5;
  const pool = document.getElementById('song-pool');
  const slotList = document.getElementById('slot-list');
  const submitBtn = document.getElementById('submit-btn');
  if (!pool || !slotList || !submitBtn) return;

  // ranks[i] = song ID in slot (i+1), or null
  const ranks = Array(SLOT_COUNT).fill(null);

  const songRows = new Map();
  pool.querySelectorAll('.song-row').forEach((el) => {
    const id = parseInt(el.dataset.id, 10);
    songRows.set(id, el);
    el.addEventListener('click', (e) => {
      if (e.target.closest('a')) return;
      pickSong(id);
    });
  });

  function pickSong(id) {
    if (ranks.includes(id)) return;
    const slotIdx = ranks.indexOf(null);
    if (slotIdx === -1) return;
    ranks[slotIdx] = id;
    render();
  }

  function removeAt(idx) {
    ranks.splice(idx, 1);
    ranks.push(null);
    render();
  }

  function move(idx, delta) {
    const target = idx + delta;
    if (target < 0 || target >= SLOT_COUNT) return;
    if (ranks[target] === null && delta > 0) return;
    [ranks[idx], ranks[target]] = [ranks[target], ranks[idx]];
    render();
  }

  function render() {
    // Pool: hide rows whose IDs are in ranks
    songRows.forEach((el, id) => {
      el.classList.toggle('is-hidden', ranks.includes(id));
    });

    // Slots
    slotList.innerHTML = '';
    for (let i = 0; i < SLOT_COUNT; i++) {
      const slot = document.createElement('li');
      slot.className = 'slot';
      const songId = ranks[i];
      if (songId !== null) slot.classList.add('is-filled');

      const rank = document.createElement('span');
      rank.className = 'slot-rank';
      rank.textContent = String(i + 1);
      slot.appendChild(rank);

      if (songId === null) {
        const empty = document.createElement('span');
        empty.className = 'slot-empty-label';
        empty.textContent = 'pick a song';
        slot.appendChild(empty);
      } else {
        const row = songRows.get(songId);
        const meta = document.createElement('div');
        meta.className = 'song-meta';
        const title = document.createElement('div');
        title.className = 'song-title';
        title.textContent = row.dataset.title;
        const artist = document.createElement('div');
        artist.className = 'song-artist';
        artist.textContent = row.dataset.artist;
        meta.append(title, artist);
        slot.appendChild(meta);

        const controls = document.createElement('div');
        controls.className = 'slot-controls';
        controls.append(
          mkBtn('▲', () => move(i, -1), i === 0, 'Move up'),
          mkBtn('▼', () => move(i, +1), i === SLOT_COUNT - 1 || ranks[i + 1] === null, 'Move down'),
          mkBtn('×', () => removeAt(i), false, 'Remove', 'btn-remove'),
        );
        slot.appendChild(controls);
      }
      slotList.appendChild(slot);
    }

    const filled = ranks.filter((r) => r !== null).length;
    submitBtn.disabled = filled !== SLOT_COUNT;
  }

  function mkBtn(label, onClick, disabled, title, extraClass) {
    const b = document.createElement('button');
    b.type = 'button';
    b.textContent = label;
    b.title = title;
    b.disabled = !!disabled;
    if (extraClass) b.classList.add(extraClass);
    b.addEventListener('click', onClick);
    return b;
  }

  submitBtn.addEventListener('click', async () => {
    if (ranks.includes(null)) return;
    const payload = { ranks };
    console.log('submitting ballot', payload);
    try {
      const res = await fetch('/vote', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
      console.log('server response', res.status);
      alert('submitted (M1 stub — not persisted)');
    } catch (err) {
      console.error('submit failed', err);
      alert('submit failed: ' + err.message);
    }
  });

  render();
})();
