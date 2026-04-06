<script setup lang="ts">
const router = useRouter()

const token = ref('')
const errorMsg = ref('')
const loading = ref(false)

async function handleLogin() {
  if (!token.value.trim()) {
    errorMsg.value = 'Token is required'
    return
  }

  loading.value = true
  errorMsg.value = ''

  try {
    const res = await fetch('/api/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ token: token.value }),
    })

    if (res.ok) {
      router.push('/')
    } else {
      const data = await res.json().catch(() => null)
      errorMsg.value = data?.error ?? 'Authentication failed'
    }
  } catch {
    errorMsg.value = 'Network error — is the server running?'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-page">
    <form class="login-card" @submit.prevent="handleLogin">
      <h1 class="login-title">Juli</h1>
      <p class="login-subtitle">Tmux Web Console</p>

      <div class="form-group">
        <label for="token-input" class="form-label">Access Token</label>
        <input
          id="token-input"
          v-model="token"
          type="password"
          class="form-input"
          placeholder="Enter your token"
          autocomplete="current-password"
          :disabled="loading"
        />
      </div>

      <div v-if="errorMsg" class="error-msg">
        {{ errorMsg }}
      </div>

      <button type="submit" class="login-btn" :disabled="loading">
        {{ loading ? 'Connecting...' : 'Connect' }}
      </button>
    </form>
  </div>
</template>

<style scoped>
.login-page {
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #0d1117;
}

.login-card {
  width: 100%;
  max-width: 360px;
  padding: 40px;
  background: #161b22;
  border: 1px solid #21262d;
  border-radius: 8px;
}

.login-title {
  margin: 0;
  font-size: 32px;
  font-weight: 700;
  color: #58a6ff;
  text-align: center;
  letter-spacing: -0.03em;
}

.login-subtitle {
  margin: 4px 0 32px;
  font-size: 14px;
  color: #484f58;
  text-align: center;
}

.form-group {
  margin-bottom: 20px;
}

.form-label {
  display: block;
  margin-bottom: 6px;
  font-size: 12px;
  font-weight: 600;
  color: #c9d1d9;
}

.form-input {
  width: 100%;
  padding: 10px 12px;
  background: #0d1117;
  border: 1px solid #21262d;
  border-radius: 4px;
  color: #c9d1d9;
  font-family: inherit;
  font-size: 14px;
  outline: none;
  transition: border-color 0.15s;
  box-sizing: border-box;
}

.form-input:focus {
  border-color: #58a6ff;
}

.form-input:disabled {
  opacity: 0.6;
}

.error-msg {
  margin-bottom: 16px;
  padding: 8px 12px;
  background: #3d1117;
  border: 1px solid #ff7b7233;
  border-radius: 4px;
  color: #ff7b72;
  font-size: 13px;
}

.login-btn {
  width: 100%;
  padding: 10px;
  background: #1f6feb;
  border: none;
  border-radius: 4px;
  color: #ffffff;
  font-family: inherit;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s;
}

.login-btn:hover:not(:disabled) {
  background: #388bfd;
}

.login-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
</style>
