let currentUser = JSON.parse(localStorage.getItem('user')) || null;
let activeModalPostId = null;
let currentLoadedPosts = [];

function getDefaultAvatar(seed) {
  return `https://api.dicebear.com/7.x/bottts/svg?seed=${encodeURIComponent(seed || 'corstellae')}`;
}

const sidebar = document.getElementById('sidebar');
const toggleSidebar = document.getElementById('toggleSidebar');
const closeSidebar = document.getElementById('closeSidebar');
const sidebarOverlay = document.getElementById('sidebarOverlay');
const navItems = document.querySelectorAll('.nav-item');
const viewSections = document.querySelectorAll('.view-section');
const pageTitle = document.getElementById('pageTitle');

// Lightbox & Comments Modals
const imageModal = document.getElementById('imageModal');
const modalImage = document.getElementById('modalImage');
const closeModalBtn = document.getElementById('closeModalBtn');

const commentsModal = document.getElementById('commentsModal');
const closeCommentsModalBtn = document.getElementById('closeCommentsModalBtn');
const modalCommentsList = document.getElementById('modalCommentsList');
const modalCommentInput = document.getElementById('modalCommentInput');
const modalCommentSubmitBtn = document.getElementById('modalCommentSubmitBtn');

closeModalBtn?.addEventListener('click', () => imageModal.style.display = 'none');
closeCommentsModalBtn?.addEventListener('click', () => commentsModal.style.display = 'none');

function openImageModal(src) {
  modalImage.src = src;
  imageModal.style.display = 'flex';
}

function initUserUI() {
  if (!currentUser) return;

  const fallbackPic = getDefaultAvatar(currentUser.username || currentUser.nickname);
  document.getElementById('sideName').textContent = currentUser.nickname || currentUser.username || 'User';
  document.getElementById('sideAvatar').src = currentUser.avatar_url || fallbackPic;

  const badge = document.getElementById('sideRoleBadge');
  if (currentUser.role === 'admin') {
    badge.textContent = '👑 Admin';
    badge.className = 'badge badge-admin';
    document.getElementById('adminTabBtn').style.display = 'flex';
  } else if (currentUser.role === 'moderator') {
    badge.textContent = '🛡️ Mod';
    badge.className = 'badge badge-mod';
    document.getElementById('adminTabBtn').style.display = 'flex';
  } else {
    badge.textContent = 'User';
    badge.className = 'badge';
    document.getElementById('adminTabBtn').style.display = 'none';
  }
}

// Sidebar Drawer
toggleSidebar?.addEventListener('click', () => {
  sidebar.classList.add('open');
  sidebarOverlay.classList.add('open');
});

closeSidebar?.addEventListener('click', closeMenu);
sidebarOverlay?.addEventListener('click', closeMenu);

function closeMenu() {
  sidebar.classList.remove('open');
  sidebarOverlay.classList.remove('open');
}

// Navigation
navItems.forEach(item => {
  item.addEventListener('click', () => {
    const targetView = item.getAttribute('data-view');

    navItems.forEach(n => n.classList.remove('active'));
    item.classList.add('active');

    viewSections.forEach(sec => {
      sec.style.display = sec.id === targetView ? 'block' : 'none';
    });

    if (targetView === 'feedView') {
      pageTitle.textContent = 'Dashboard Feed';
      loadPublicPosts();
    } else if (targetView === 'profileView') {
      pageTitle.textContent = 'User Profile';
      loadProfileData();
    } else if (targetView === 'adminView') {
      pageTitle.textContent = 'Admin & Moderation Panel';
      loadAdminPanel();
    }

    closeMenu();
  });
});

// Image Upload Preview
const imageInput = document.getElementById('imageInput');
const postPreviewContainer = document.getElementById('postPreviewContainer');
const postImagePreview = document.getElementById('postImagePreview');
const removePostImageBtn = document.getElementById('removePostImageBtn');

imageInput?.addEventListener('change', (e) => {
  const file = e.target.files[0];
  if (file) {
    const reader = new FileReader();
    reader.onload = (event) => {
      postImagePreview.src = event.target.result;
      postPreviewContainer.style.display = 'flex';
    };
    reader.readAsDataURL(file);
  }
});

removePostImageBtn?.addEventListener('click', () => {
  imageInput.value = '';
  postPreviewContainer.style.display = 'none';
  postImagePreview.src = '';
});

// Load Feed Posts
async function loadPublicPosts() {
  const feedGrid = document.getElementById('feedGrid');
  try {
    const res = await fetch('/api/posts/public');
    const posts = await res.json();
    currentLoadedPosts = posts || [];
    feedGrid.innerHTML = '';

    if (!posts || posts.length === 0) {
      feedGrid.innerHTML = `<div class="feed-card"><p>No public posts yet.</p></div>`;
      return;
    }

    posts.forEach(p => {
      let badgeHtml = '';
      if (!p.is_anonymous) {
        if (p.author_role === 'admin') badgeHtml = '<span class="badge badge-admin">👑 Admin</span>';
        else if (p.author_role === 'moderator') badgeHtml = '<span class="badge badge-mod">🛡️ Mod</span>';
      }

      const canManagePost = currentUser && (currentUser.role === 'admin' || currentUser.role === 'moderator' || currentUser.id === p.user_id);
      const postAvatar = p.author_pic || getDefaultAvatar(p.author_name);

      const commentCount = p.comments ? p.comments.length : 0;

      const card = document.createElement('div');
      card.className = 'feed-card';
      card.innerHTML = `
        <div class="post-header">
          <div class="post-author">
            <img src="${postAvatar}" alt="Avatar" />
            <div>
              <strong>${p.author_name}</strong> ${badgeHtml}
            </div>
          </div>
          ${canManagePost ? `
            <div class="post-management">
              <button onclick="promptEditPost(${p.id}, '${p.content.replace(/'/g, "\\'")}')" class="btn-sm btn-primary">Edit</button>
              <button onclick="deletePost(${p.id})" class="btn-sm btn-logout">Delete</button>
            </div>
          ` : ''}
        </div>

        <p class="post-text">${p.content}</p>
        ${p.image_url ? `<img src="${p.image_url}" class="post-image" onclick="openImageModal('${p.image_url}')" style="cursor:pointer;" alt="Attachment" />` : ''}

        <div class="comments-section">
          <button onclick="openCommentsModal(${p.id})" class="view-comments-btn">
            💬 ${commentCount > 0 ? `View all ${commentCount} comment(s)` : 'Write a comment...'}
          </button>
        </div>
      `;
      feedGrid.appendChild(card);
    });
  } catch (err) {
    feedGrid.innerHTML = `<p>Error loading posts.</p>`;
  }
}

// Facebook-Style Scrollable Comments Modal Logic
function openCommentsModal(postId) {
  activeModalPostId = postId;
  const post = currentLoadedPosts.find(p => p.id === postId);
  renderModalComments(post ? post.comments : []);
  commentsModal.style.display = 'flex';
}

function renderModalComments(comments) {
  modalCommentsList.innerHTML = '';
  if (!comments || comments.length === 0) {
    modalCommentsList.innerHTML = '<p class="sub-text">No comments yet. Be the first to comment!</p>';
    return;
  }

  comments.forEach(c => {
    let cBadge = '';
    if (c.author_role === 'admin') cBadge = '<span class="badge badge-admin">👑 Admin</span>';
    else if (c.author_role === 'moderator') cBadge = '<span class="badge badge-mod">🛡️ Mod</span>';

    const canDelete = currentUser && (currentUser.role === 'admin' || currentUser.role === 'moderator' || currentUser.id === c.user_id);
    const commentAvatar = c.author_pic || getDefaultAvatar(c.author_name);

    const div = document.createElement('div');
    div.className = 'comment-item';
    div.innerHTML = `
      <img src="${commentAvatar}" class="comment-avatar" alt="Avatar" />
      <div class="comment-body">
        <strong>${c.author_name}</strong> ${cBadge}
        <p>${c.content}</p>
      </div>
      ${canDelete ? `<button onclick="deleteCommentModal(${c.id})" class="btn-sm btn-logout">Delete</button>` : ''}
    `;
    modalCommentsList.appendChild(div);
  });
  modalCommentsList.scrollTop = modalCommentsList.scrollHeight;
}

modalCommentSubmitBtn?.addEventListener('click', async () => {
  if (!activeModalPostId || !modalCommentInput.value.trim()) return;

  await fetch('/api/comments/create', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      post_id: activeModalPostId,
      user_id: currentUser ? currentUser.id : 1,
      content: modalCommentInput.value.trim()
    }),
  });

  modalCommentInput.value = '';
  await loadPublicPosts();
  const updatedPost = currentLoadedPosts.find(p => p.id === activeModalPostId);
  renderModalComments(updatedPost ? updatedPost.comments : []);
});

async function deleteCommentModal(commentId) {
  if (!confirm('Delete this comment?')) return;

  await fetch('/api/comments/delete', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ comment_id: commentId }),
  });

  await loadPublicPosts();
  const updatedPost = currentLoadedPosts.find(p => p.id === activeModalPostId);
  renderModalComments(updatedPost ? updatedPost.comments : []);
}

// Post Submission Handler
document.getElementById('postForm')?.addEventListener('submit', async (e) => {
  e.preventDefault();
  const uploadStatus = document.getElementById('uploadStatus');
  const postContent = document.getElementById('postContent');

  uploadStatus.textContent = 'Submitting post...';
  let imageUrl = '';

  if (imageInput.files.length > 0) {
    const formData = new FormData();
    formData.append('image', imageInput.files[0]);

    try {
      const uploadRes = await fetch('/api/upload', { method: 'POST', body: formData });
      const uploadData = await uploadRes.json();
      if (uploadData.url) imageUrl = uploadData.url;
    } catch (err) {
      uploadStatus.textContent = 'Failed to upload photo.';
      return;
    }
  }

  try {
    const res = await fetch('/api/posts/create', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        user_id: currentUser ? currentUser.id : 1,
        content: postContent.value,
        image_url: imageUrl,
        is_anonymous: document.getElementById('anonCheckbox').checked,
      }),
    });
    const data = await res.json();

    alert(data.message);
    document.getElementById('postForm').reset();
    postPreviewContainer.style.display = 'none';
    uploadStatus.textContent = '';
    loadPublicPosts();
  } catch (err) {
    uploadStatus.textContent = 'Error submitting post.';
  }
});

async function promptEditPost(postId, currentContent) {
  const newContent = prompt('Edit your post:', currentContent);
  if (!newContent || newContent.trim() === '') return;

  await fetch('/api/posts/edit', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ post_id: postId, content: newContent }),
  });
  loadPublicPosts();
}

async function deletePost(postId) {
  if (!confirm('Are you sure you want to delete this post?')) return;

  await fetch('/api/posts/delete', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ post_id: postId }),
  });
  loadPublicPosts();
}

// Profile Handler
let pendingAvatarUrl = null;

function loadProfileData() {
  if (!currentUser) return;
  pendingAvatarUrl = currentUser.avatar_url;
  document.getElementById('editNickname').value = currentUser.nickname || '';
  document.getElementById('editBio').value = currentUser.bio || '';
  
  const fallbackPic = getDefaultAvatar(currentUser.username || currentUser.nickname);
  document.getElementById('profileAvatarPreview').src = currentUser.avatar_url || fallbackPic;
}

document.getElementById('avatarInput')?.addEventListener('change', (e) => {
  const file = e.target.files[0];
  if (file) {
    const reader = new FileReader();
    reader.onload = (event) => {
      document.getElementById('profileAvatarPreview').src = event.target.result;
    };
    reader.readAsDataURL(file);
  }
});

document.getElementById('removeAvatarBtn')?.addEventListener('click', () => {
  pendingAvatarUrl = '';
  document.getElementById('avatarInput').value = '';
  const fallbackPic = getDefaultAvatar(currentUser ? currentUser.username : 'default');
  document.getElementById('profileAvatarPreview').src = fallbackPic;
});

document.getElementById('profileForm')?.addEventListener('submit', async (e) => {
  e.preventDefault();
  const avatarFile = document.getElementById('avatarInput').files[0];
  let avatarUrl = pendingAvatarUrl;

  if (avatarFile) {
    const formData = new FormData();
    formData.append('image', avatarFile);
    const uploadRes = await fetch('/api/upload', { method: 'POST', body: formData });
    const uploadData = await uploadRes.json();
    if (uploadData.url) avatarUrl = uploadData.url;
  }

  const updatedData = {
    user_id: currentUser.id,
    nickname: document.getElementById('editNickname').value,
    bio: document.getElementById('editBio').value,
    avatar_url: avatarUrl,
  };

  await fetch('/api/user/update', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(updatedData),
  });

  currentUser.nickname = updatedData.nickname;
  currentUser.bio = updatedData.bio;
  currentUser.avatar_url = updatedData.avatar_url;
  localStorage.setItem('user', JSON.stringify(currentUser));
  initUserUI();
  alert('Profile updated successfully!');
});

// Admin Moderation Panel
async function loadAdminPanel() {
  const pendingRes = await fetch('/api/admin/pending');
  const pendingPosts = await pendingRes.json();
  const pendingList = document.getElementById('pendingPostsList');
  pendingList.innerHTML = '';

  if (!pendingPosts || pendingPosts.length === 0) {
    pendingList.innerHTML = '<p class="sub-text">No pending posts awaiting review.</p>';
  } else {
    pendingPosts.forEach(p => {
      const item = document.createElement('div');
      item.className = 'admin-item';
      item.innerHTML = `
        <div class="admin-item-content">
          <div><strong>${p.author_name}</strong>: ${p.content}</div>
          ${p.image_url ? `<img src="${p.image_url}" class="admin-thumb" onclick="openImageModal('${p.image_url}')" title="Click to enlarge" />` : ''}
        </div>
        <div class="admin-item-actions">
          <button onclick="reviewPost(${p.id}, 'approved')" class="btn-sm btn-primary">Approve</button>
          <button onclick="reviewPost(${p.id}, 'rejected')" class="btn-sm btn-logout">Reject</button>
        </div>
      `;
      pendingList.appendChild(item);
    });
  }

  const usersRes = await fetch('/api/admin/users');
  const users = await usersRes.json();
  const usersList = document.getElementById('usersList');
  usersList.innerHTML = '';

  (users || []).forEach(u => {
    const item = document.createElement('div');
    item.className = 'admin-item';
    item.innerHTML = `
      <div>
        <strong>${u.nickname}</strong> (@${u.username}) - <em>${u.role}</em>
      </div>
      <div class="admin-item-actions">
        ${currentUser && currentUser.role === 'admin' ? `
          <button onclick="toggleRole(${u.id}, '${u.role === 'moderator' ? 'user' : 'moderator'}')" class="btn-sm btn-primary">
            ${u.role === 'moderator' ? 'Remove Mod' : 'Make Mod'}
          </button>
          <button onclick="toggleBan(${u.id},${!u.is_banned})" class="btn-sm btn-logout">
            ${u.is_banned ? 'Unban' : 'Ban'}
          </button>
        ` : ''}
      </div>
    `;
    usersList.appendChild(item);
  });
}

async function reviewPost(postId, status) {
  await fetch('/api/admin/review', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ post_id: postId, status }),
  });
  loadAdminPanel();
}

async function toggleRole(userId, role) {
  await fetch('/api/admin/user/role', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ user_id: userId, role }),
  });
  loadAdminPanel();
}

async function toggleBan(userId, isBanned) {
  await fetch('/api/admin/user/ban', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ user_id: userId, is_banned: isBanned }),
  });
  loadAdminPanel();
}

// Fixed Logout: Invalidate 7-Day Session Cookie & Clear Local Storage
document.getElementById('logoutBtn')?.addEventListener('click', async () => {
  try {
    await fetch('/api/logout', { method: 'POST' });
  } catch (e) {}
  localStorage.removeItem('user');
  window.location.href = '/';
});

initUserUI();
loadPublicPosts();