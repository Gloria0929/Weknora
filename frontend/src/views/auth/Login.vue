<template>
  <div class="auth-page">
    <!-- 左侧品牌面板 -->
    <aside class="brand-panel">

      <div class="brand-panel__inner">
        <div class="brand-panel__eyebrow">KNOWLEDGE / AGENT / WORKSPACE</div>

        <div class="brand-copy">
          <h1 class="brand-copy__title">{{ $t("platform.subtitle") }}</h1>
          <p class="brand-copy__desc">{{ $t("platform.description") }}</p>
        </div>

        <ul class="brand-features">
          <li class="brand-features__item">
            <span class="brand-features__check">
              <svg
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="3"
              >
                <polyline points="20 6 9 17 4 12" />
              </svg>
            </span>
            <span>{{ $t("platform.multimodalParsing") }}</span>
          </li>
          <li class="brand-features__item">
            <span class="brand-features__check">
              <svg
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="3"
              >
                <polyline points="20 6 9 17 4 12" />
              </svg>
            </span>
            <span>{{ $t("platform.hybridSearchEngine") }}</span>
          </li>
          <li class="brand-features__item">
            <span class="brand-features__check">
              <svg
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="3"
              >
                <polyline points="20 6 9 17 4 12" />
              </svg>
            </span>
            <span>{{ $t("platform.ragQandA") }}</span>
          </li>
        </ul>

        <div class="brand-tags">
          <span class="brand-tag">{{ $t("platform.rag") }}</span>
          <span class="brand-tag">{{ $t("platform.agent") }}</span>
          <span class="brand-tag">{{ $t("platform.wiki") }}</span>
          <span class="brand-tag">{{ $t("platform.hybridSearch") }}</span>
        </div>

        <div class="brand-panel__meta">
          <span class="brand-panel__meta-dot"></span>
          <span>Private knowledge, ready to work</span>
        </div>
      </div>
    </aside>

    <!-- 右侧表单区 -->
    <main class="form-panel">
      <div class="form-panel__inner">
        <!-- 登录 -->
        <div v-if="!isRegisterMode" class="auth-card">
          <div v-if="inviteLookup" class="invite-banner">
            <t-icon name="link" class="invite-banner__icon" />
            <div class="invite-banner__text">
              <div class="invite-banner__title">
                {{
                  $t("inviteRegister.bannerTitle", {
                    tenant: inviteLookup.tenant_name || "",
                  })
                }}
              </div>
              <div class="invite-banner__hint">
                {{ $t("inviteRegister.bannerHintLogin") }}
              </div>
            </div>
          </div>
          <div
            v-else-if="inviteLookupError"
            class="invite-banner invite-banner--error"
          >
            {{ inviteLookupError }}
          </div>

          <header class="auth-card__header">
            <div class="auth-card__eyebrow">WEKNORA WORKSPACE</div>
            <h2 class="auth-card__title">{{ $t("auth.login") }}</h2>
            <p class="auth-card__subtitle">{{ $t("auth.subtitle") }}</p>
          </header>

          <p v-if="registrationEnabled" class="auth-card__hint">
            {{ $t("auth.loginHint") }}
          </p>

          <t-form
            ref="formRef"
            :data="formData"
            :rules="formRules"
            @submit="handleLogin"
            layout="vertical"
            label-align="top"
            class="auth-form"
          >
            <t-form-item :label="$t('auth.email')" name="email">
              <t-input
                v-model="formData.email"
                :placeholder="$t('auth.emailPlaceholder')"
                type="text"
                autocomplete="email"
                size="large"
                :disabled="loading"
              />
            </t-form-item>

            <t-form-item :label="$t('auth.password')" name="password">
              <t-input
                v-model="formData.password"
                :placeholder="$t('auth.passwordPlaceholder')"
                type="password"
                autocomplete="current-password"
                size="large"
                :disabled="loading"
                @enter="handleLogin"
              />
            </t-form-item>

            <t-button
              theme="default"
              type="submit"
              size="large"
              block
              :loading="loading"
              class="btn btn--primary"
            >
              {{ loading ? $t("auth.loggingIn") : $t("auth.login") }}
            </t-button>

            <t-button
              v-if="registrationEnabled"
              theme="default"
              variant="outline"
              size="large"
              block
              :disabled="loading"
              class="btn btn--ghost"
              @click="toggleMode"
            >
              {{ $t("auth.createAccount") }}
            </t-button>

            <template v-if="oidcEnabled">
              <div class="auth-divider">
                <span>{{ $t("auth.orContinueWith") }}</span>
              </div>
              <t-button
                theme="default"
                size="large"
                block
                :loading="oidcLoading"
                :disabled="loading"
                class="btn btn--oidc"
                @click="handleOIDCLogin"
              >
                {{ oidcLoading ? $t("auth.redirectingToOIDC") : oidcLoginText }}
              </t-button>
            </template>
          </t-form>
        </div>

        <!-- 注册 -->
        <div v-else class="auth-card">
          <div v-if="inviteLookup" class="invite-banner">
            <t-icon name="link" class="invite-banner__icon" />
            <div class="invite-banner__text">
              <div class="invite-banner__title">
                {{
                  $t("inviteRegister.bannerTitle", {
                    tenant: inviteLookup.tenant_name || "",
                  })
                }}
              </div>
              <div class="invite-banner__hint">
                {{ $t("inviteRegister.bannerHint") }}
              </div>
            </div>
          </div>
          <div
            v-else-if="inviteLookupError"
            class="invite-banner invite-banner--error"
          >
            {{ inviteLookupError }}
          </div>

          <header class="auth-card__header">
            <div class="auth-card__eyebrow">WEKNORA WORKSPACE</div>
            <h2 class="auth-card__title">{{ $t("auth.createAccount") }}</h2>
            <p class="auth-card__subtitle">{{ $t("auth.subtitle") }}</p>
          </header>

          <t-form
            ref="registerFormRef"
            :data="registerData"
            :rules="registerRules"
            @submit="handleRegister"
            layout="vertical"
            label-align="top"
            class="auth-form"
          >
            <t-form-item :label="$t('auth.username')" name="username">
              <t-input
                v-model="registerData.username"
                :placeholder="$t('auth.usernamePlaceholder')"
                size="large"
                :disabled="loading"
              />
            </t-form-item>

            <t-form-item :label="$t('auth.email')" name="email">
              <t-input
                v-model="registerData.email"
                :placeholder="$t('auth.emailPlaceholder')"
                type="text"
                autocomplete="email"
                size="large"
                :disabled="loading"
              />
            </t-form-item>

            <t-form-item :label="$t('auth.password')" name="password">
              <t-input
                v-model="registerData.password"
                :placeholder="$t('auth.passwordPlaceholder')"
                type="password"
                autocomplete="new-password"
                size="large"
                :disabled="loading"
              />
            </t-form-item>

            <t-form-item
              :label="$t('auth.confirmPassword')"
              name="confirmPassword"
            >
              <t-input
                v-model="registerData.confirmPassword"
                :placeholder="$t('auth.confirmPasswordPlaceholder')"
                type="password"
                autocomplete="new-password"
                size="large"
                :disabled="loading"
                @enter="handleRegister"
              />
            </t-form-item>

            <t-button
              theme="default"
              type="submit"
              size="large"
              block
              :loading="loading"
              class="btn btn--primary"
            >
              {{ loading ? $t("auth.registering") : $t("auth.register") }}
            </t-button>
          </t-form>

          <div class="auth-card__footer">
            <span>{{ $t("auth.haveAccount") }}</span>
            <a href="#" @click.prevent="toggleMode" class="auth-link">
              {{ $t("auth.backToLogin") }}
            </a>
          </div>
        </div>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, nextTick, onMounted, computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import { MessagePlugin } from "tdesign-vue-next";
import { useRoleLabel } from "@/composables/useRoleLabel";
import { notifyLoginSuccess } from "@/utils/loginNotify";
import { newPasswordRules } from "@/utils/passwordPolicy";
import {
  login,
  register,
  getOIDCAuthorizationURL,
  getOIDCConfig,
  autoSetup,
  getAuthConfig,
  userInfoFromApi,
  getInvitationByToken,
  registerByInvite,
  type InviteLookup,
} from "@/api/auth";
import { useAuthStore } from "@/stores/auth";
import { useI18n } from "vue-i18n";

const router = useRouter();
const route = useRoute();
const authStore = useAuthStore();
const { t, tm } = useI18n();
const { formatRole, roleIcon } = useRoleLabel();

const formRef = ref();
const registerFormRef = ref();

const loading = ref(false);
const oidcLoading = ref(false);
const isRegisterMode = ref(false);
const oidcEnabled = ref(false);
const oidcProviderName = ref("");
const registrationEnabled = ref(true);
const complexPasswordEnabled = ref(false);

const inviteToken = ref("");
const inviteLookup = ref<InviteLookup | null>(null);
const inviteLookupError = ref("");
const inviteLookupLoading = ref(false);

const oidcLoginText = computed(() => {
  if (oidcProviderName.value) {
    return t("auth.oidcLoginWithProvider", {
      provider: oidcProviderName.value,
    });
  }
  return t("auth.oidcLogin");
});

const formData = reactive<{ [key: string]: any }>({
  email: "",
  password: "",
});

const registerData = reactive<{ [key: string]: any }>({
  username: "",
  email: "",
  password: "",
  confirmPassword: "",
});

const formRules = computed(() => ({
  email: [
    { required: true, message: t("auth.emailRequired"), type: "error" },
    { email: true, message: t("auth.emailInvalid"), type: "error" },
  ],
  password: [
    { required: true, message: t("auth.passwordRequired"), type: "error" },
    { min: 8, message: t("auth.passwordMinLength"), type: "error" },
    { max: 32, message: t("auth.passwordMaxLength"), type: "error" },
  ],
}));

const registerRules = computed(() => ({
  username: [
    { required: true, message: t("auth.usernameRequired"), type: "error" },
    { min: 2, message: t("auth.usernameMinLength"), type: "error" },
    { max: 20, message: t("auth.usernameMaxLength"), type: "error" },
    {
      pattern: /^[a-zA-Z0-9_\u4e00-\u9fa5]+$/,
      message: t("auth.usernameInvalid"),
      type: "error",
    },
  ],
  email: [
    { required: true, message: t("auth.emailRequired"), type: "error" },
    { email: true, message: t("auth.emailInvalid"), type: "error" },
  ],
  password: newPasswordRules(t, complexPasswordEnabled.value),
  confirmPassword: [
    {
      required: true,
      message: t("auth.confirmPasswordRequired"),
      type: "error",
    },
    {
      validator: (val: string) => val === registerData.password,
      message: t("auth.passwordMismatch"),
      type: "error",
    },
  ],
}));

const toggleMode = () => {
  isRegisterMode.value = !isRegisterMode.value;
  Object.keys(registerData).forEach((key) => {
    (registerData as any)[key] = "";
  });
};

const persistLoginResponse = async (response: any, skipRedirect = false) => {
  const activeTenant = response.active_tenant || response.tenant;
  if (response.user && response.token) {
    const homeTenantIdRaw = response.user.tenant_id ?? activeTenant?.id ?? "";
    authStore.setUser(userInfoFromApi(response.user, homeTenantIdRaw));
    authStore.setToken(response.token);
    if (response.refresh_token) {
      authStore.setRefreshToken(response.refresh_token);
    }
    if (activeTenant) {
      authStore.setTenant({
        id: String(activeTenant.id) || "",
        name: activeTenant.name || "",
        owner_id: response.user.id || "",
        created_at: activeTenant.created_at || new Date().toISOString(),
        updated_at: activeTenant.updated_at || new Date().toISOString(),
      });
    } else {
      authStore.setTenant(null);
    }
    if (Array.isArray(response.memberships)) {
      authStore.setMemberships(response.memberships);
    }
    const activeIdNum = Number(activeTenant?.id);
    const homeIdNum = Number(homeTenantIdRaw);
    if (
      Number.isFinite(activeIdNum) &&
      Number.isFinite(homeIdNum) &&
      activeIdNum !== homeIdNum
    ) {
      authStore.setSelectedTenant(activeIdNum, activeTenant?.name || null);
    } else {
      authStore.setSelectedTenant(null, null);
    }
  }

  await authStore.refreshFromAuthMe();
  await nextTick();
  if (skipRedirect) return;
  router.replace(
    authStore.hasValidTenant
      ? "/platform/creatChat"
      : "/onboarding/workspace",
  );
};

const getBackendOIDCRedirectURI = () =>
  `${window.location.origin}/api/v1/auth/oidc/callback`;

const loadOIDCConfig = async () => {
  try {
    const response = await getOIDCConfig();
    oidcEnabled.value = !!response.success && !!response.enabled;
    oidcProviderName.value = response.provider_display_name || "";
  } catch {
    oidcEnabled.value = false;
    oidcProviderName.value = "";
  }
};

const loadAuthConfig = async () => {
  try {
    const response = await getAuthConfig();
    registrationEnabled.value = response.registration_mode !== "invite_only";
    complexPasswordEnabled.value = response.complex_password_enabled;
  } catch {
    registrationEnabled.value = true;
    complexPasswordEnabled.value = false;
  }
};

const handleOIDCLogin = async () => {
  try {
    oidcLoading.value = true;
    const response = await getOIDCAuthorizationURL(getBackendOIDCRedirectURI());
    const authorizationURL = response.authorization_url;

    if (!response.success || !authorizationURL) {
      MessagePlugin.error(response.message || t("auth.oidcLoginFailed"));
      return;
    }

    if (inviteToken.value) {
      sessionStorage.setItem("weknora_pending_invite_token", inviteToken.value);
    }
    window.location.href = authorizationURL;
  } catch (error: any) {
    console.error("OIDC 登录跳转失败:", error);
    MessagePlugin.error(error.message || t("auth.oidcLoginFailed"));
  } finally {
    oidcLoading.value = false;
  }
};

const acceptAndEnter = async (token: string) => {
  loading.value = true;
  try {
    const result = await authStore.acceptInvitationByTokenAndRefresh(token);
    if (result.ok) {
      MessagePlugin.success(t("inviteRegister.joined"));
    } else {
      MessagePlugin.warning(t("inviteRegister.invalidBody"));
    }
  } catch {
    MessagePlugin.warning(t("inviteRegister.invalidBody"));
  } finally {
    loading.value = false;
    await nextTick();
    router.replace("/platform/creatChat");
  }
};

const handleLogin = async () => {
  try {
    const valid = await formRef.value?.validate();
    if (valid !== true) return;

    loading.value = true;

    const response = await login({
      email: formData.email,
      password: formData.password,
    });

    if (response.success) {
      if (inviteToken.value) {
        await persistLoginResponse(response, true);
        await acceptAndEnter(inviteToken.value);
        return;
      }
      await persistLoginResponse(response);
      notifyLoginSuccess(response, t, tm, formatRole, roleIcon);
    } else {
      MessagePlugin.error(response.message || t("auth.loginError"));
    }
  } catch (error: any) {
    console.error("登录错误:", error);
    MessagePlugin.error(error.message || t("auth.loginErrorRetry"));
  } finally {
    loading.value = false;
  }
};

const handleRegister = async () => {
  try {
    const valid = await registerFormRef.value?.validate();
    if (valid !== true) return;

    loading.value = true;

    if (inviteToken.value) {
      const response = await registerByInvite({
        token: inviteToken.value,
        username: registerData.username,
        email: registerData.email,
        password: registerData.password,
      });
      if (!response.success) {
        MessagePlugin.error(response.message || t("auth.registerFailed"));
        return;
      }
      MessagePlugin.success(t("auth.registerSuccess"));
      await persistLoginResponse(response);
      return;
    }

    const response = await register({
      username: registerData.username,
      email: registerData.email,
      password: registerData.password,
    });

    if (response.success) {
      MessagePlugin.success(t("auth.registerSuccess"));
      isRegisterMode.value = false;
      formData.email = registerData.email;
      Object.keys(registerData).forEach((key) => {
        (registerData as any)[key] = "";
      });
    } else {
      MessagePlugin.error(response.message || t("auth.registerFailed"));
    }
  } catch (error: any) {
    console.error("注册错误:", error);
    MessagePlugin.error(error.message || t("auth.registerError"));
  } finally {
    loading.value = false;
  }
};

onMounted(async () => {
  const tokenFromQuery = String(route.query.token || "").trim();
  if (tokenFromQuery) {
    inviteToken.value = tokenFromQuery;
    inviteLookupLoading.value = true;
    try {
      const resp = await getInvitationByToken(tokenFromQuery);
      if (resp.success && resp.data) {
        inviteLookup.value = resp.data;
      } else {
        inviteLookupError.value =
          resp.message || t("inviteRegister.invalidBody");
        loadOIDCConfig();
        loadAuthConfig();
        return;
      }
    } catch {
      inviteLookupError.value = t("inviteRegister.invalidBody");
      loadOIDCConfig();
      loadAuthConfig();
      return;
    } finally {
      inviteLookupLoading.value = false;
    }

    if (authStore.isLoggedIn && (await authStore.refreshFromAuthMe())) {
      await acceptAndEnter(tokenFromQuery);
      return;
    }

    const cfg = await getAuthConfig();
    const inviteOnly = cfg.registration_mode === "invite_only";
    registrationEnabled.value = !inviteOnly;
    isRegisterMode.value = !inviteOnly;
    loadOIDCConfig();
    return;
  }

  if (authStore.isLoggedIn) {
    router.replace("/platform/creatChat");
    return;
  }

  const AUTO_SETUP_FAILED_KEY = "weknora_auto_setup_failed";
  if (localStorage.getItem(AUTO_SETUP_FAILED_KEY) !== "true") {
    try {
      const response = await autoSetup();
      if (response.success) {
        authStore.setLiteMode(true);
        await persistLoginResponse(response);
        return;
      } else {
        localStorage.setItem(AUTO_SETUP_FAILED_KEY, "true");
      }
    } catch {
      localStorage.setItem(AUTO_SETUP_FAILED_KEY, "true");
    }
  }

  loadOIDCConfig();
  loadAuthConfig();
});
</script>

<style lang="less" scoped>
/* ---------- 页面骨架 ---------- */
.auth-page {
  display: flex;
  min-height: 100dvh;
  width: 100%;
  background: var(--td-bg-color-page);
  font-family: var(--app-font-family);
  overflow: hidden;
}

/* The entry screen shares the welcome page's paper and charcoal surfaces. */
.brand-panel {
  position: relative;
  flex: 0 0 46%;
  min-width: 0;
  background: var(--td-bg-color-sidebar);
  color: var(--td-text-color-primary);
  border-right: 1px solid var(--td-component-stroke);
  display: flex;
  align-items: center;
  padding: 44px clamp(32px, 5vw, 76px);
  box-sizing: border-box;
}

.brand-panel__inner {
  position: relative;
  z-index: 2;
  width: 100%;
  max-width: 460px;
  display: flex;
  flex-direction: column;
  min-height: calc(100dvh - 88px);
}

.brand-logo {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 22px;
}

.brand-logo__mark {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 38px;
  height: 38px;
  border-radius: var(--app-radius-xl);
  background: var(--td-brand-color-light);
  color: var(--td-text-color-primary);
  font-weight: 800;
  font-size: var(--app-text-4xl);
  letter-spacing: -0.7px;
  box-shadow: 0 0 0 6px color-mix(in srgb, var(--td-brand-color-light) 8%, transparent);
}

.brand-logo__text {
  font-size: var(--app-text-4xl);
  font-weight: 700;
  letter-spacing: -0.3px;
  color: var(--td-text-color-primary);
}

.brand-panel__eyebrow,
.auth-card__eyebrow {
  font-size: var(--app-text-xs);
  line-height: 1;
  letter-spacing: 0.16em;
  font-weight: 700;
}

.brand-panel__eyebrow {
  color: var(--td-text-color-secondary);
  margin-bottom: 74px;
}

.brand-copy {
  margin-bottom: 42px;
}

.brand-copy__title {
  font-size: clamp(34px, 4vw, 48px);
  font-family: var(--app-font-family-display);
  line-height: 1.3;
  font-weight: 500;
  letter-spacing: -1.5px;
  margin: 0 0 18px 0;
  color: var(--td-text-color-primary);
  text-wrap: balance;
}

.brand-copy__desc {
  font-size: var(--app-text-xl);
  line-height: 1.7;
  color: var(--td-text-color-secondary);
  margin: 0;
  max-width: 390px;
}

.brand-features {
  list-style: none;
  padding: 0;
  margin: 0 0 40px 0;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.brand-features__item {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: var(--app-text-lg);
  color: var(--td-text-color-secondary);
}

.brand-features__check {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  background: color-mix(in srgb, var(--td-brand-color) 22%, transparent);
  color: var(--td-brand-color-hover);
  flex-shrink: 0;

  svg {
    width: 12px;
    height: 12px;
  }
}

.brand-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: auto;
  margin-bottom: 20px;
}

.brand-tag {
  display: inline-block;
  padding: 7px 12px;
  border-radius: var(--app-radius-md);
  border: 1px solid var(--td-component-stroke);
  background: var(--td-bg-color-container);
  font-size: var(--app-text-md);
  font-weight: 500;
  color: var(--td-text-color-secondary);
}

.brand-panel__meta {
  display: flex;
  align-items: center;
  gap: 9px;
  font-size: var(--app-text-sm);
  color: var(--td-text-color-secondary);
}

.brand-panel__meta-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--td-brand-color-hover);
  box-shadow: 0 0 0 4px color-mix(in srgb, var(--td-brand-color-hover) 12%, transparent);
}

.brand-footer {
  margin-top: 48px;
  font-size: var(--app-text-md);
  color: var(--td-text-color-secondary);
}

/* ---------- 右侧表单区（亮暗色自适应）---------- */
.form-panel {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  padding: 48px clamp(24px, 6vw, 96px);
  box-sizing: border-box;
  background: var(--td-bg-color-page);

  &::after {
    content: none;
    position: absolute;
    right: 8%;
    bottom: 10%;
    width: 160px;
    height: 160px;
    border: 1px solid var(--td-component-stroke);
    border-radius: var(--app-radius-xl);
    transform: rotate(18deg);
    pointer-events: none;
  }
}

.form-panel__inner {
  width: 100%;
  max-width: 430px;
  position: relative;
  z-index: 1;
}

.auth-card {
  width: 100%;
  padding: 40px 42px 36px;
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-2xl);
  background: var(--td-bg-color-container);
  box-shadow: var(--app-surface-shadow);
  box-sizing: border-box;
}

.auth-card__header {
  margin-bottom: 26px;
}

.auth-card__eyebrow {
  color: var(--td-brand-color);
  margin-bottom: 18px;
}

.auth-card__title {
  font-family: var(--app-font-family-display);
  font-size: 30px;
  font-weight: 500;
  letter-spacing: -1px;
  color: var(--td-text-color-primary);
  margin: 0 0 8px 0;
  line-height: 1.25;
}

.auth-card__subtitle {
  font-size: var(--app-text-lg);
  color: var(--td-text-color-secondary);
  margin: 0;
  line-height: 1.5;
}

.auth-card__hint {
  margin: 0 0 24px 0;
  padding: 10px 14px;
  border-radius: var(--app-radius-xl);
  background: var(--td-bg-color-secondarycontainer);
  border: 1px solid var(--td-component-stroke);
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-md);
  line-height: 1.5;
}

.auth-card__footer {
  margin-top: 24px;
  text-align: center;
  font-size: var(--app-text-base);
  color: var(--td-text-color-secondary);
}

.auth-link {
  color: var(--td-brand-color);
  text-decoration: none;
  font-weight: 600;
  margin-left: 4px;

  &:hover {
    text-decoration: underline;
  }
}

/* ---------- 表单 ---------- */
.auth-form {
  :deep(.t-form-item__label) {
    font-size: var(--app-text-base);
    font-weight: 500;
    color: var(--td-text-color-primary);
    margin-bottom: 6px;
    padding: 0;
  }

  :deep(.t-form-item) {
    margin-bottom: 20px;
  }

  :deep(.t-input) {
    border-radius: var(--app-radius-xl);
    border-color: var(--td-component-border);
    background: var(--td-bg-color-container);
    transition:
      border-color var(--app-motion-base) ease,
      box-shadow 0.18s ease;

    &:hover {
      border-color: var(--td-text-color-placeholder);
    }

    &:focus-within {
      border-color: var(--td-brand-color);
      box-shadow: 0 0 0 3px var(--td-brand-color-focus);
    }
  }

  :deep(.t-input__inner) {
    font-size: var(--app-text-lg);
  }
}

.btn {
  height: 50px;
  border-radius: var(--app-radius-xl);
  font-size: var(--app-text-lg);
  font-weight: 600;
  transition: all var(--app-motion-base) ease;

  & + .btn {
    margin-top: 12px;
  }
}

.btn--primary {
  background: var(--td-text-color-primary) !important;
  border-color: var(--td-text-color-primary) !important;
  border: none;
  color: var(--td-bg-color-container) !important;
  margin-top: 30px;

  &:hover,
  &:focus {
    opacity: 0.88;
    color: var(--td-bg-color-container) !important;
    transform: translateY(-1px);
  }

  &:active {
    opacity: 0.78;
    color: var(--td-bg-color-container) !important;
    transform: translateY(0);
  }
}

.btn--ghost {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-border);
  color: var(--td-text-color-primary);

  &:hover {
    border-color: var(--td-brand-color);
    color: var(--td-brand-color);
    background: var(--td-bg-color-secondarycontainer);
  }
}

.btn--oidc {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-border);
  color: var(--td-text-color-secondary);

  &:hover {
    border-color: var(--td-text-color-placeholder);
    background: var(--td-bg-color-secondarycontainer-hover);
  }
}

.auth-divider {
  position: relative;
  text-align: center;
  margin: 22px 0 18px;
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-md);

  span {
    position: relative;
    z-index: 1;
    padding: 0 12px;
    background: var(--td-bg-color-container);
  }

  &::before {
    content: "";
    position: absolute;
    left: 0;
    right: 0;
    top: 50%;
    border-top: 1px solid var(--td-component-stroke);
  }
}

/* ---------- 邀请提示 ---------- */
.invite-banner {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 12px 14px;
  margin-bottom: 24px;
  border-radius: var(--app-radius-lg);
  background: var(--td-bg-color-secondarycontainer);
  border: 1px solid var(--td-component-stroke);
}

.invite-banner__icon {
  margin-top: 2px;
  font-size: var(--app-text-2xl);
  flex-shrink: 0;
  color: var(--td-brand-color);
}

.invite-banner__text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.invite-banner__title {
  font-size: var(--app-text-base);
  font-weight: 600;
  line-height: 1.4;
  color: var(--td-text-color-primary);
}

.invite-banner__hint {
  font-size: var(--app-text-md);
  color: var(--td-text-color-secondary);
  line-height: 1.5;
}

.invite-banner--error {
  background: var(--td-error-color-light);
  border-color: color-mix(in srgb, var(--td-error-color) 30%, transparent);
  color: var(--td-error-color);
  font-size: var(--app-text-base);
}

/* ---------- 响应式 ---------- */
@media (max-width: 960px) {
  .auth-page {
    flex-direction: column;
  }

  .brand-panel {
    flex: 0 0 auto;
    min-width: 0;
    padding: 36px 32px 40px;
  }

  .brand-panel__inner {
    min-height: 0;
  }

  .brand-logo {
    margin-bottom: 18px;
  }

  .brand-panel__eyebrow {
    margin-bottom: 42px;
  }

  .brand-copy {
    margin-bottom: 28px;

    &__title {
      font-size: var(--app-text-4xl);
    }
  }

  .brand-features {
    margin-bottom: 24px;
  }

  .brand-footer {
    display: none;
  }

  .form-panel {
    padding: 48px 24px;
    align-items: flex-start;
  }

  .auth-card {
    padding: 34px 32px 30px;
  }
}

@media (max-width: 520px) {
  .brand-panel {
    padding: 32px 20px;
  }

  .brand-panel__inner {
    min-height: auto;
  }

  .brand-panel__eyebrow {
    margin-bottom: 34px;
  }

  .brand-copy__title {
    font-size: var(--app-text-4xl);
  }

  .brand-copy__desc {
    font-size: var(--app-text-base);
  }

  .form-panel {
    padding: 32px 20px;
  }

  .auth-card__title {
    font-size: var(--app-text-4xl);
  }

  .auth-card {
    padding: 30px 22px 26px;
    border-radius: var(--app-radius-xl);
  }
}
</style>
