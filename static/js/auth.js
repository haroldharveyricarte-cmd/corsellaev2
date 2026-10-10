const tabRegister = document.getElementById('tabRegister');
const tabLogin = document.getElementById('tabLogin');
const registerForm = document.getElementById('registerForm');
const loginForm = document.getElementById('loginForm');
const authMsg = document.getElementById('authMsg');

tabRegister.addEventListener('click', () => {
  tabRegister.classList.add('active');
  tabLogin.classList.remove('active');
  registerForm.style.display = 'block';
  loginForm.style.display = 'none';
  authMsg.textContent = '';
});

tabLogin.addEventListener('click', () => {
  tabLogin.classList.add('active');
  tabRegister.classList.remove('active');
  registerForm.style.display = 'none';
  loginForm.style.display = 'block';
  authMsg.textContent = '';
});

// Handle Signup
registerForm.addEventListener('submit', async (e) => {
  e.preventDefault();
  authMsg.textContent = 'Creating account...';

  const body = {
    username: document.getElementById('regUsername').value.trim(),
    nickname: document.getElementById('regNickname').value.trim(),
    email: document.getElementById('regEmail').value.trim(),
    password: document.getElementById('regPassword').value,
  };

  try {
    const res = await fetch('/api/register', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    const data = await res.json();
    authMsg.textContent = data.message;

    if (res.ok) {
      setTimeout(() => {
        tabLogin.click();
      }, 1200);
    }
  } catch (err) {
    authMsg.textContent = 'Server error during registration.';
  }
});

// Handle Login
loginForm.addEventListener('submit', async (e) => {
  e.preventDefault();
  authMsg.textContent = 'Logging in...';

  const body = {
    email_or_user: document.getElementById('loginIdentifier').value.trim(),
    password: document.getElementById('loginPassword').value,
  };

  try {
    const res = await fetch('/api/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    const data = await res.json();

    if (res.ok && data.user) {
      authMsg.textContent = 'Login successful! Redirecting...';
      localStorage.setItem('user', JSON.stringify(data.user));
      setTimeout(() => {
        window.location.href = '/dashboard';
      }, 800);
    } else {
      authMsg.textContent = data.message || 'Invalid credentials.';
    }
  } catch (err) {
    authMsg.textContent = 'Server error during login.';
  }
});