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

// Modals
const imageModal = document.getElementById('imageModal');
const modalImage = document.getElementById('modalImage');
const closeModalBtn = document.getElementById('closeModalBtn');

const commentsModal = document.getElementById('commentsModal');
const closeCommentsModalBtn = document.getElementById('closeCommentsModalBtn');
const modalCommentsList = document.getElementById('modalCommentsList');
const modalCommentInput = document.getElementById('modalCommentInput');
const modalCommentSubmitBtn = document.getElementById('modalCommentSubmitBtn');

const reactionUsersModal = document.getElementById('reactionUsersModal');
const closeReactionUsersModalBtn = document.getElementById('closeReactionUsersModalBtn');
const reactionUsersList = document.getElementById('reactionUsersList');

closeModalBtn?.addEventListener('click', () => imageModal.style.display = 'none');
closeCommentsModalBtn?.addEventListener('click', () => commentsModal.style.display = 'none');
closeReactionUsersModalBtn?.addEventListener('click', () => reactionUsersModal.style.display = 'none');

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

  // Check if the logged-in user is you (haroldharvey / developer)
  if (currentUser.username === 'haroldharvey' || currentUser.role === 'developer') {
    badge.innerHTML = `
      <span class="badge-dev-icon" style="display: inline-flex; align-items: center;">
        <svg viewBox="0 0 100 100" xmlns="http://www.w3.org/2000/svg" style="width: 14px; height: 14px; vertical-align: middle;">
          <defs>
            <linearGradient id="devGrad" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" stop-color="#a7eaff" />
              <stop offset="50%" stop-color="#b6b7f9" />
              <stop offset="100%" stop-color="#fec6ff" />
            </linearGradient>
          </defs>
          <path class="hex-bg" d="M50 2.5L93.3 27.5V72.5L50 97.5L6.7 72.5V27.5L50 2.5Z" fill="url(#devGrad)" stroke="#ffffff" stroke-width="4"/>
          <path class="code-symbol" d="M38.5 35L26 50L38.5 65M61.5 35L74 50L61.5 65M47.5 70L52.5 30" stroke="white" stroke-width="9" stroke-linecap="round" fill="none"/>
        </svg>
      </span> Developer
    `;
    badge.className = 'badge badge-dev';
    document.getElementById('adminTabBtn').style.display = 'flex'; // Grants admin panel access
  } else if (currentUser.role === 'admin') {
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

  loadNotifications();
  // Poll notifications every 5 seconds so multi-account sessions update live
  setInterval(loadNotifications, 5000);
}

// Drawer Controls
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

function showCommentToast(message = "Your comment was submitted and is pending admin approval!") {
    const toast = document.getElementById('toastNotification');
    if (!toast) return;
    
    toast.textContent = message;
    toast.style.display = 'block';
    toast.style.opacity = '1';

    // Auto hide after 4 seconds
    setTimeout(() => {
        toast.style.opacity = '0';
        setTimeout(() => {
            toast.style.display = 'none';
        }, 300);
    }, 4000);
}

// Navigation
navItems.forEach(item => {
  item.addEventListener('click', () => {
    const targetView = item.getAttribute('data-view');
    switchView(targetView);
  });
});

function switchView(targetView) {
  navItems.forEach(n => {
    if (n.getAttribute('data-view') === targetView) n.classList.add('active');
    else n.classList.remove('active');
  });

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
}

// Search Bar Logic
const searchInput = document.getElementById('searchInput');
const clearSearchBtn = document.getElementById('clearSearchBtn');

searchInput?.addEventListener('input', (e) => {
  const query = e.target.value.trim();
  clearSearchBtn.style.display = query ? 'block' : 'none';
  loadPublicPosts(query);
});

clearSearchBtn?.addEventListener('click', () => {
  searchInput.value = '';
  clearSearchBtn.style.display = 'none';
  loadPublicPosts();
});

// Notifications Dropdown
const notifBellBtn = document.getElementById('notifBellBtn');
const notifDropdown = document.getElementById('notifDropdown');
const markReadBtn = document.getElementById('markReadBtn');

notifBellBtn?.addEventListener('click', () => {
  const isVisible = notifDropdown.style.display === 'block';
  notifDropdown.style.display = isVisible ? 'none' : 'block';
});

async function loadNotifications() {
  try {
    const res = await fetch('/api/notifications');
    const notifs = await res.json();
    const list = document.getElementById('notifList');
    const badge = document.getElementById('notifBadge');
    list.innerHTML = '';

    let unreadCount = 0;
    (notifs || []).forEach(n => {
      if (!n.is_read) unreadCount++;
      const div = document.createElement('div');
      div.className = `notif-item ${n.is_read ? '' : 'unread'}`;

      // Only add click action if post_id exists and is greater than 0
      if (n.post_id && n.post_id > 0) {
        div.onclick = () => handleNotifClick(n.post_id);
        div.style.cursor = 'pointer';
      } else {
        div.style.cursor = 'default';
      }

      // Fallback logic for message text if empty
      let displayMessage = n.message;
      if (!displayMessage || displayMessage.trim() === '') {
        if (n.type === 'post_rejected') displayMessage = 'Your post was reviewed and rejected by an admin.';
        else if (n.type === 'comment_rejected') displayMessage = 'Your comment was reviewed and rejected by an admin.';
        else if (n.type === 'post_approved') displayMessage = 'Your post was approved by an admin!';
        else if (n.type === 'comment_approved') displayMessage = 'Your comment was approved by an admin!';
        else displayMessage = 'sent a notification';
      }

      div.innerHTML = `
        <img src="${n.actor_pic || getDefaultAvatar(n.actor_name)}" alt="Avatar" />
        <div>
          <strong>${n.actor_name}</strong> ${displayMessage}
        </div>
      `;
      list.appendChild(div);
    });

    if (unreadCount > 0) {
      badge.textContent = unreadCount;
      badge.style.display = 'inline-block';
    } else {
      badge.style.display = 'none';
    }
  } catch (err) {
    console.error("Error loading notifications:", err);
  }
}

async function handleNotifClick(postId) {
  notifDropdown.style.display = 'none';
  switchView('feedView');
  await loadPublicPosts();

  setTimeout(() => {
    const targetCard = document.getElementById(`post-${postId}`);
    if (targetCard) {
      targetCard.scrollIntoView({ behavior: 'smooth', block: 'center' });
      targetCard.classList.add('highlighted-post');
      setTimeout(() => targetCard.classList.remove('highlighted-post'), 2500);
    }
  }, 300);
}

markReadBtn?.addEventListener('click', async () => {
  await fetch('/api/notifications/read', { method: 'POST' });
  loadNotifications();
});


// Image Upload Previews for Post Creation
const imageInput = document.getElementById('imageInput');
const postPreviewContainer = document.getElementById('postPreviewContainer');
const postImagePreview = document.getElementById('postImagePreview');
const removePostImageBtn = document.getElementById('removePostImageBtn');

imageInput?.addEventListener('change', function(e) {
  const file = e.target.files[0];
  if (file) {
    const reader = new FileReader();
    reader.onload = function(event) {
      postImagePreview.src = event.target.result;
      postPreviewContainer.style.display = 'flex';
    };
    reader.readAsDataURL(file);
  }
});

removePostImageBtn?.addEventListener('click', function() {
  imageInput.value = '';
  postPreviewContainer.style.display = 'none';
  postImagePreview.src = '';
});

// Image Upload Previews for Profile
const avatarInput = document.getElementById('avatarInput');
const profileAvatarPreview = document.getElementById('profileAvatarPreview');

avatarInput?.addEventListener('change', function(e) {
  const file = e.target.files[0];
  if (file) {
    const reader = new FileReader();
    reader.onload = function(event) {
      profileAvatarPreview.src = event.target.result;
    };
    reader.readAsDataURL(file);
  }
});

// Load Feed Posts
async function loadPublicPosts(searchQuery = '') {
  const feedGrid = document.getElementById('feedGrid');
  try {
    const res = await fetch(`/api/posts/public${searchQuery ? '?q=' + encodeURIComponent(searchQuery) : ''}`);
    const posts = await res.json();
    currentLoadedPosts = posts || [];
    feedGrid.innerHTML = '';

    if (!posts || posts.length === 0) {
      feedGrid.innerHTML = `<div class="feed-card"><p>No public posts match your search.</p></div>`;
      return;
    }

    posts.forEach(p => {
      let badgeHtml = '';
      if (!p.is_anonymous) {
        if (p.author_username === 'haroldharvey' || p.author_role === 'developer') {
          badgeHtml = `
            <span class="badge badge-dev" title="Website Developer" style="display: inline-flex; align-items: center; gap: 4px; cursor: help;">
              <svg viewBox="0 0 100 100" xmlns="http://www.w3.org/2000/svg" style="width: 12px; height: 12px; vertical-align: middle;">
                <defs>
                  <linearGradient id="devGradFeed" x1="0%" y1="0%" x2="100%" y2="100%">
                    <stop offset="0%" stop-color="#a7eaff" />
                    <stop offset="50%" stop-color="#b6b7f9" />
                    <stop offset="100%" stop-color="#fec6ff" />
                  </linearGradient>
                </defs>
                <path d="M50 2.5L93.3 27.5V72.5L50 97.5L6.7 72.5V27.5L50 2.5Z" fill="url(#devGradFeed)" stroke="#ffffff" stroke-width="4"/>
                <path d="M38.5 35L26 50L38.5 65M61.5 35L74 50L61.5 65M47.5 70L52.5 30" stroke="white" stroke-width="9" stroke-linecap="round" fill="none"/>
              </svg> Developer
            </span>
          `;
        } else if (p.author_role === 'admin') {
          badgeHtml = '<span class="badge badge-admin" title="Website Admin" style="cursor: help;">👑 Admin</span>';
        } else if (p.author_role === 'moderator') {
          badgeHtml = '<span class="badge badge-mod" title="Website Moderator" style="cursor: help;">🛡️ Mod</span>';
        }
      }

      // Check if logged-in user has staff powers (developer, admin, or moderator)
      const isStaff = currentUser && (currentUser.role === 'developer' || currentUser.role === 'admin' || currentUser.role === 'moderator');
      const canManagePost = isStaff || (currentUser && currentUser.id === p.user_id && p.status !== 'approved');

      const postAvatar = p.author_pic || getDefaultAvatar(p.author_name);
      const commentCount = p.comments ? p.comments.length : 0;

      let reactionsHtml = `
        <div class="reactions-bar">
          <button class="reaction-btn" onclick="toggleReaction(${p.id}, null, '👍')">👍</button>
          <button class="reaction-btn" onclick="toggleReaction(${p.id}, null, '❤️')">❤️</button>
          <button class="reaction-btn" onclick="toggleReaction(${p.id}, null, '😂')">😂</button>
          <button class="reaction-btn" onclick="toggleReaction(${p.id}, null, '🔥')">🔥</button>
          ${renderReactionPills('post', p.id, p.reactions)}
        </div>
      `;

      const card = document.createElement('div');
      card.className = 'feed-card';
      card.id = `post-${p.id}`;
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

        ${reactionsHtml}

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

function renderReactionPills(type, targetId, reactions) {
  if (!reactions || reactions.length === 0) return '';
  let html = '';
  reactions.forEach(r => {
    html += `<span class="reaction-btn ${r.user_reacted ? 'active' : ''}">${r.emoji} ${r.count}</span>`;
  });
  html += `<button class="reaction-users-btn" onclick="openReactionUsersModal('${type}', ${targetId})">Who reacted?</button>`;
  return html;
}

// Instant Optimistic Reactions
async function toggleReaction(postId, commentId, emoji) {
  if (!currentUser) return;

  // Optimistic UI update on active memory post
  if (postId) {
    const post = currentLoadedPosts.find(p => p.id === postId);
    if (post) {
      if (!post.reactions) post.reactions = [];
      let r = post.reactions.find(x => x.emoji === emoji);
      if (r) {
        if (r.user_reacted) {
          r.count--;
          r.user_reacted = false;
        } else {
          r.count++;
          r.user_reacted = true;
        }
      } else {
        post.reactions.push({ emoji, count: 1, user_reacted: true });
      }
      post.reactions = post.reactions.filter(x => x.count > 0);
    }
  }

  fetch('/api/reactions/toggle', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      user_id: currentUser.id,
      post_id: postId,
      comment_id: commentId,
      emoji: emoji
    })
  });

  loadPublicPosts(searchInput.value.trim());
}

async function openReactionUsersModal(type, targetId) {
  const url = type === 'post' ? `/api/reactions/users?post_id=${targetId}` : `/api/reactions/users?comment_id=${targetId}`;
  const res = await fetch(url);
  const users = await res.json();

  reactionUsersList.innerHTML = '';
  if (!users || users.length === 0) {
    reactionUsersList.innerHTML = '<p class="sub-text">No reactions yet.</p>';
  } else {
    users.forEach(u => {
      const div = document.createElement('div');
      div.className = 'reaction-user-item';
      div.innerHTML = `
        <img src="${u.avatar_url || getDefaultAvatar(u.username)}" alt="Avatar" />
        <div><strong>${u.nickname}</strong> (@${u.username})</div>
        <span style="margin-left:auto;">${u.emoji}</span>
      `;
      reactionUsersList.appendChild(div);
    });
  }
  reactionUsersModal.style.display = 'flex';
}

// Comments Modal & Auto-Close Logic
let replyingToCommentId = null;

function openCommentsModal(postId) {
  activeModalPostId = postId;
  replyingToCommentId = null;
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
    // Check for developer badge first
    if (c.author_username === 'haroldharvey' || c.author_role === 'developer') {
      cBadge = `
        <span class="badge badge-dev" title="Website Developer" style="display: inline-flex; align-items: center; gap: 4px; cursor: help;">
          <svg viewBox="0 0 100 100" xmlns="http://www.w3.org/2000/svg" style="width: 12px; height: 12px; vertical-align: middle;">
            <defs>
              <linearGradient id="devGradComment" x1="0%" y1="0%" x2="100%" y2="100%">
                <stop offset="0%" stop-color="#a7eaff" />
                <stop offset="50%" stop-color="#b6b7f9" />
                <stop offset="100%" stop-color="#fec6ff" />
              </linearGradient>
            </defs>
            <path d="M50 2.5L93.3 27.5V72.5L50 97.5L6.7 72.5V27.5L50 2.5Z" fill="url(#devGradComment)" stroke="#ffffff" stroke-width="4"/>
            <path d="M38.5 35L26 50L38.5 65M61.5 35L74 50L61.5 65M47.5 70L52.5 30" stroke="white" stroke-width="9" stroke-linecap="round" fill="none"/>
          </svg> Developer
        </span>
      `;
    } else if (c.author_role === 'admin') {
      cBadge = '<span class="badge badge-admin" title="Website Admin" style="cursor: help;">👑 Admin</span>';
    } else if (c.author_role === 'moderator') {
      cBadge = '<span class="badge badge-mod" title="Website Moderator" style="cursor: help;">🛡️ Mod</span>';
    }

    const isStaff = currentUser && (currentUser.role === 'developer' || currentUser.role === 'admin' || currentUser.role === 'moderator');
    const canDelete = isStaff || (currentUser && currentUser.id === c.user_id);
    const commentAvatar = c.author_pic || getDefaultAvatar(c.author_name);

    let repliesHtml = '';
    if (c.replies && c.replies.length > 0) {
      c.replies.forEach(rep => {
        const canDelRep = isStaff || (currentUser && currentUser.id === rep.user_id);
        
        // Optional: Add badges to replies as well if needed
        let repBadge = '';
        if (rep.author_username === 'haroldharvey' || rep.author_role === 'developer') {
          repBadge = '<span class="badge badge-dev" title="Website Developer" style="cursor: help;">💻 Developer</span>';
        } else if (rep.author_role === 'admin') {
          repBadge = '<span class="badge badge-admin" title="Website Admin" style="cursor: help;">👑 Admin</span>';
        } else if (rep.author_role === 'moderator') {
          repBadge = '<span class="badge badge-mod" title="Website Moderator" style="cursor: help;">🛡️ Mod</span>';
        }

        repliesHtml += `
          <div class="reply-item">
            <div class="comment-item">
              <img src="${rep.author_pic || getDefaultAvatar(rep.author_name)}" class="comment-avatar" alt="Avatar" />
              <div class="comment-body">
                <strong>${rep.author_name}</strong> ${repBadge}
                <p>${rep.content}</p>
              </div>
              ${canDelRep ? `<button onclick="deleteCommentModal(${rep.id})" class="btn-sm btn-logout">Delete</button>` : ''}
            </div>
          </div>
        `;
      });
    }

    const div = document.createElement('div');
    div.className = 'comment-item-group';
    div.innerHTML = `
      <div class="comment-item">
        <img src="${commentAvatar}" class="comment-avatar" alt="Avatar" />
        <div class="comment-body">
          <strong>${c.author_name}</strong> ${cBadge}
          <p>${c.content}</p>
          <div class="reactions-bar">
            <button class="reaction-btn" onclick="toggleReaction(null, ${c.id}, '👍')">👍</button>
            <button class="reaction-btn" onclick="toggleReaction(null, ${c.id}, '❤️')">❤️</button>
            <button class="view-comments-btn" onclick="setReplyTarget(${c.id}, '${c.author_name}')">Reply</button>
            ${renderReactionPills('comment', c.id, c.reactions)}
          </div>
        </div>
        ${canDelete ? `<button onclick="deleteCommentModal(${c.id})" class="btn-sm btn-logout">Delete</button>` : ''}
      </div>
      ${repliesHtml}
    `;
    modalCommentsList.appendChild(div);
  });
}

function setReplyTarget(commentId, authorName) {
  replyingToCommentId = commentId;
  modalCommentInput.placeholder = `Replying to ${authorName}...`;
  modalCommentInput.focus();
}

modalCommentSubmitBtn?.addEventListener('click', async () => {
  if (!activeModalPostId || !modalCommentInput.value.trim()) return;

  const contentText = modalCommentInput.value.trim();
  modalCommentInput.value = '';
  
  // Close the comment modal immediately for a smooth experience
  commentsModal.style.display = 'none';

  const res = await fetch('/api/comments/create', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      post_id: activeModalPostId,
      user_id: currentUser ? currentUser.id : 1,
      parent_id: replyingToCommentId,
      content: contentText
    }),
  });

  if (res.ok) {
    // Trigger the confirmation popup toast right here!
    showCommentToast("💬 Your comment was submitted and is pending admin approval.");
  }

  replyingToCommentId = null;
  modalCommentInput.placeholder = 'Write a comment...';
  loadPublicPosts(searchInput.value.trim());
});

async function deleteCommentModal(commentId) {
  if (!confirm('Delete this comment?')) return;

  await fetch('/api/comments/delete', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ comment_id: commentId }),
  });

  await loadPublicPosts(searchInput.value.trim());
  const updatedPost = currentLoadedPosts.find(p => p.id === activeModalPostId);
  renderModalComments(updatedPost ? updatedPost.comments : []);
}

// Post Submissions
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

    // Replaced native alert with nice toast notification
    showCommentToast(data.message || "🚀 Post submitted for review!");
    
    document.getElementById('postForm').reset();
    postPreviewContainer.style.display = 'none';
    uploadStatus.textContent = '';
    loadPublicPosts();
  } catch (err) {
    uploadStatus.textContent = 'Error submitting post.';
  }
});

async function promptEditPost(postId, currentContent) {
  // Replaced native prompt with styled custom modal prompt
  showCustomPrompt('Edit Post', currentContent, async (newContent) => {
    const res = await fetch('/api/posts/edit', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ user_id: currentUser ? currentUser.id : 1, post_id: postId, content: newContent }),
    });
    const data = await res.json();
    if (data.status === 'error') {
      showCommentToast(data.message);
    } else {
      showCommentToast('✅ Post updated successfully!');
    }
    loadPublicPosts();
  });
}

async function deletePost(postId) {
  // Replaced native confirm with styled custom confirmation modal
  showCustomConfirm('Delete Post', 'Are you sure you want to delete this post? This action cannot be undone.', async () => {
    const res = await fetch('/api/posts/delete', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ user_id: currentUser ? currentUser.id : 1, post_id: postId }),
    });
    const data = await res.json();
    if (data.status === 'error') {
      showCommentToast(data.message);
    } else {
      showCommentToast('🗑️ Post deleted successfully.');
    }
    loadPublicPosts();
  });
}

// Custom Confirmation Dialog Helper
function showCustomConfirm(title, message, onConfirm) {
  const modal = document.getElementById('customConfirmModal');
  const titleEl = document.getElementById('confirmModalTitle');
  const msgEl = document.getElementById('confirmModalMessage');
  const inputEl = document.getElementById('confirmModalInput');
  const okBtn = document.getElementById('confirmModalOkBtn');
  const cancelBtn = document.getElementById('confirmModalCancelBtn');

  titleEl.textContent = title;
  msgEl.textContent = message;
  inputEl.style.display = 'none'; // Hide input for normal confirms
  modal.style.display = 'flex';

  // Cleanup previous listeners
  const newOkBtn = okBtn.cloneNode(true);
  const newCancelBtn = cancelBtn.cloneNode(true);
  okBtn.parentNode.replaceChild(newOkBtn, okBtn);
  cancelBtn.parentNode.replaceChild(newCancelBtn, cancelBtn);

  newOkBtn.addEventListener('click', () => {
    modal.style.display = 'none';
    onConfirm();
  });

  newCancelBtn.addEventListener('click', () => {
    modal.style.display = 'none';
  });
}

// Custom Prompt Dialog Helper (for editing posts)
function showCustomPrompt(title, initialValue, onConfirm) {
  const modal = document.getElementById('customConfirmModal');
  const titleEl = document.getElementById('confirmModalTitle');
  const msgEl = document.getElementById('confirmModalMessage');
  const inputEl = document.getElementById('confirmModalInput');
  const okBtn = document.getElementById('confirmModalOkBtn');
  const cancelBtn = document.getElementById('confirmModalCancelBtn');

  titleEl.textContent = title;
  msgEl.textContent = 'Modify your post content below:';
  inputEl.value = initialValue;
  inputEl.style.display = 'block'; // Show textarea input
  modal.style.display = 'flex';
  inputEl.focus();

  const newOkBtn = okBtn.cloneNode(true);
  const newCancelBtn = cancelBtn.cloneNode(true);
  okBtn.parentNode.replaceChild(newOkBtn, okBtn);
  cancelBtn.parentNode.replaceChild(newCancelBtn, cancelBtn);

  newOkBtn.addEventListener('click', () => {
    const val = inputEl.value.trim();
    if (!val) return;
    modal.style.display = 'none';
    onConfirm(val);
  });

  newCancelBtn.addEventListener('click', () => {
    modal.style.display = 'none';
  });
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

// Admin Moderation Panel & Tab Management
function loadAdminPanel() {
    loadPendingPosts();
    loadPendingComments();
    loadUsersList();
}

function switchAdminTab(tab) {
    const postsBtn = document.getElementById('tab-posts-btn');
    const commentsBtn = document.getElementById('tab-comments-btn');
    const postsContainer = document.getElementById('admin-posts-container');
    const commentsContainer = document.getElementById('admin-comments-container');

    if (tab === 'posts') {
        postsBtn.style.background = '#4f46e5';
        postsBtn.style.color = 'white';
        commentsBtn.style.background = 'transparent';
        commentsBtn.style.color = '#9ca3af';
        postsContainer.style.display = 'block';
        commentsContainer.style.display = 'none';
        loadPendingPosts();
    } else {
        commentsBtn.style.background = '#4f46e5';
        commentsBtn.style.color = 'white';
        postsBtn.style.background = 'transparent';
        postsBtn.style.color = '#9ca3af';
        commentsContainer.style.display = 'block';
        postsContainer.style.display = 'none';
        loadPendingComments();
    }
}

async function loadPendingPosts() {
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
}

async function loadPendingComments() {
    const commentsRes = await fetch('/api/admin/comments/pending');
    const pendingComments = await commentsRes.json();
    const commentsList = document.getElementById('pending-comments-list');
    commentsList.innerHTML = '';

    if (!pendingComments || pendingComments.length === 0) {
        commentsList.innerHTML = '<p class="sub-text" style="text-align: center; color: #9ca3af; padding: 20px;">✨ No pending comments awaiting review.</p>';
    } else {
        pendingComments.forEach(c => {
            const item = document.createElement('div');
            item.style.cssText = "background: #111827; border: 1px solid #374151; padding: 14px; border-radius: 10px; margin-bottom: 10px; display: flex; justify-content: space-between; align-items: center; gap: 12px;";
            item.innerHTML = `
                <div style="display: flex; align-items: flex-start; gap: 10px; overflow: hidden; flex: 1;">
                    <img src="${c.author_pic || 'https://api.dicebear.com/7.x/bottts/svg?seed=fallback'}" style="width: 32px; height: 32px; border-radius: 50%; background: #374151; object-fit: cover; flex-shrink: 0;" />
                    <div style="flex: 1; min-width: 0;">
                        <div style="display: flex; align-items: center; gap: 8px; margin-bottom: 4px;">
                            <strong style="color: #f3f4f6; font-size: 13px;">${c.author_name}</strong>
                            <span style="font-size: 10px; background: rgba(99, 102, 241, 0.2); color: #818cf8; padding: 2px 6px; border-radius: 9999px;">${c.author_role}</span>
                        </div>
                        <!-- Highlighted Comment Box -->
                        <div style="background: rgba(79, 70, 229, 0.12); border-left: 3px solid #6366f1; padding: 8px 12px; border-radius: 6px; margin: 6px 0;">
                            <p style="color: #ffffff; font-size: 14px; font-weight: 500; margin: 0; word-break: break-word;">"${c.content}"</p>
                        </div>
                        <span style="font-size: 11px; color: #9ca3af;">Target Post ID: ${c.post_id} • ${new Date(c.created_at).toLocaleString()}</span>
                    </div>
                </div>
                <div style="display: flex; gap: 6px; flex-shrink: 0;">
                    <button onclick="reviewComment(${c.id}, 'approved')" style="background: #16a34a; color: white; border: none; padding: 6px 12px; border-radius: 6px; font-size: 12px; font-weight: 600; cursor: pointer;">Approve</button>
                    <button onclick="reviewComment(${c.id}, 'rejected')" style="background: #dc2626; color: white; border: none; padding: 6px 12px; border-radius: 6px; font-size: 12px; font-weight: 600; cursor: pointer;">Reject</button>
                </div>
            `;
            commentsList.appendChild(item);
        });
    }
}

async function loadUsersList() {
    const usersRes = await fetch('/api/admin/users');
    const users = await usersRes.json();
    const usersList = document.getElementById('usersList');
    usersList.innerHTML = '';

    (users || []).forEach(u => {
        let roleDisplay = u.role;
        if (u.username === 'haroldharvey' || u.role === 'developer') {
            roleDisplay = '<span class="badge badge-dev" title="Website Developer" style="cursor: help;">💻 Developer</span>';
        } else if (u.role === 'admin') {
            roleDisplay = '<span class="badge badge-admin" title="Website Admin" style="cursor: help;">👑 Admin</span>';
        } else if (u.role === 'moderator') {
            roleDisplay = '<span class="badge badge-mod" title="Website Moderator" style="cursor: help;">🛡️ Mod</span>';
        } else {
            roleDisplay = '<span class="badge" title="Standard User">User</span>';
        }

        const item = document.createElement('div');
        item.className = 'admin-item';
        item.innerHTML = `
            <div>
                <strong>${u.nickname}</strong> (@${u.username}) - <em>${roleDisplay}</em>
            </div>
            <div class="admin-item-actions">
                ${currentUser && (currentUser.role === 'admin' || currentUser.role === 'developer') ? `
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
    loadPendingPosts();
}

async function reviewComment(commentId, status) {
    await fetch('/api/admin/comments/review', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ comment_id: commentId, status }),
    });
    loadPendingComments();
}

async function toggleRole(userId, role) {
    await fetch('/api/admin/user/role', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ user_id: userId, role }),
    });
    loadUsersList();
}

async function toggleBan(userId, isBanned) {
    await fetch('/api/admin/user/ban', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ user_id: userId, is_banned: isBanned }),
    });
    loadUsersList();
}

// Logout
document.getElementById('logoutBtn')?.addEventListener('click', async () => {
  try {
    await fetch('/api/logout', { method: 'POST' });
  } catch (e) {}
  localStorage.removeItem('user');
  window.location.href = '/';
});

initUserUI();
loadPublicPosts();