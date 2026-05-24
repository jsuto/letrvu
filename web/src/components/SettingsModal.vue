<template>
  <div v-if="visible" class="fixed inset-0 bg-black/30 z-[100] flex items-center justify-center" @click.self="close">
    <div class="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl w-[480px] flex flex-col shadow-xl max-h-[90vh] overflow-y-auto">

      <!-- Header -->
      <div class="flex justify-between items-center px-4 py-3 border-b border-[var(--color-border)] text-sm font-medium sticky top-0 bg-[var(--color-surface)] z-[1]">
        <span>{{ $t('settings.title') }}</span>
        <button @click="close" class="bg-none border-none text-lg cursor-pointer text-[var(--color-text-muted)]">×</button>
      </div>

      <!-- Body -->
      <div class="px-4 py-4 flex flex-col gap-3.5">
        <label class="flex flex-col gap-1 text-xs text-[var(--color-text-muted)]">
          {{ $t('settings.displayName') }}
          <input v-model="form.display_name" type="text" :placeholder="$t('settings.displayNamePlaceholder')"
            class="px-2.5 py-2 border border-[var(--color-border)] rounded-md text-sm bg-[var(--color-bg)] text-[var(--color-text)] outline-none focus:border-teal" />
        </label>
        <label class="flex flex-col gap-1 text-xs text-[var(--color-text-muted)]">
          {{ $t('settings.signature') }}
          <textarea v-model="form.signature" placeholder="Your name&#10;your@email.com"
            class="px-2.5 py-2 border border-[var(--color-border)] rounded-md text-sm bg-[var(--color-bg)] text-[var(--color-text)] outline-none resize-y min-h-[100px] leading-relaxed focus:border-teal" />
        </label>

        <label class="flex flex-col gap-1 text-xs text-[var(--color-text-muted)]">
          {{ $t('settings.undoSend') }}
          <select v-model.number="form.undo_send_delay"
            class="px-2.5 py-2 border border-[var(--color-border)] rounded-md text-sm bg-[var(--color-bg)] text-[var(--color-text)] outline-none focus:border-teal">
            <option :value="0">{{ $t('settings.undoOff') }}</option>
            <option :value="5">{{ $t('settings.undo5s') }}</option>
            <option :value="10">{{ $t('settings.undo10s') }}</option>
            <option :value="20">{{ $t('settings.undo20s') }}</option>
            <option :value="30">{{ $t('settings.undo30s') }}</option>
          </select>
        </label>

        <label class="flex flex-col gap-1 text-xs text-[var(--color-text-muted)]">
          {{ $t('settings.pollInterval') }}
          <select v-model.number="form.poll_interval"
            class="px-2.5 py-2 border border-[var(--color-border)] rounded-md text-sm bg-[var(--color-bg)] text-[var(--color-text)] outline-none focus:border-teal">
            <option :value="0">{{ $t('settings.pollOff') }}</option>
            <option :value="60">{{ $t('settings.poll1min') }}</option>
            <option :value="120">{{ $t('settings.poll2min') }}</option>
            <option :value="300">{{ $t('settings.poll5min') }}</option>
            <option :value="600">{{ $t('settings.poll10min') }}</option>
          </select>
        </label>

        <div class="text-xs text-[var(--color-text-muted)] font-medium pt-1 border-t border-[var(--color-border)] mt-1">{{ $t('settings.notificationsSection') }}</div>
        <div class="flex items-center gap-2.5 text-sm">
          <span class="text-[var(--color-text)] flex-1">{{ $t('settings.desktopNotifications') }}</span>
          <template v-if="notifPermission === 'denied'">
            <span class="text-xs text-[var(--color-text-muted)]">{{ $t('settings.notificationsBlocked') }}</span>
          </template>
          <template v-else-if="settings.notificationsEnabled && notifPermission === 'granted'">
            <span class="text-xs text-teal font-medium">{{ $t('settings.notificationsOn') }}</span>
            <button @click="disableNotifications"
              class="px-3 py-1.5 border border-[var(--color-border)] rounded-md bg-[var(--color-surface)] text-xs cursor-pointer text-[var(--color-text)] whitespace-nowrap hover:border-teal hover:text-teal">{{ $t('settings.disable') }}</button>
          </template>
          <template v-else>
            <button @click="enableNotifications"
              class="px-3 py-1.5 border border-[var(--color-border)] rounded-md bg-[var(--color-surface)] text-xs cursor-pointer text-[var(--color-text)] whitespace-nowrap hover:border-teal hover:text-teal">{{ $t('settings.enable') }}</button>
          </template>
        </div>

        <div class="flex items-center gap-2.5">
          <span class="text-sm text-[var(--color-text)] flex-1">{{ $t('settings.eventReminders') }}</span>
          <select v-model.number="form.calendar_reminder_minutes"
            class="px-2.5 py-2 border border-[var(--color-border)] rounded-md text-sm bg-[var(--color-bg)] text-[var(--color-text)] outline-none focus:border-teal w-auto">
            <option :value="0">{{ $t('settings.remindersOff') }}</option>
            <option :value="5">{{ $t('settings.reminders5min') }}</option>
            <option :value="10">{{ $t('settings.reminders10min') }}</option>
            <option :value="15">{{ $t('settings.reminders15min') }}</option>
            <option :value="30">{{ $t('settings.reminders30min') }}</option>
            <option :value="60">{{ $t('settings.reminders1hour') }}</option>
          </select>
        </div>

        <!-- Read receipts -->
        <div class="text-xs text-[var(--color-text-muted)] font-medium pt-1 border-t border-[var(--color-border)] mt-1">{{ $t('settings.readReceiptsSection') }}</div>
        <label class="flex items-center gap-2.5">
          <span class="text-sm text-[var(--color-text)] flex-1">{{ $t('settings.readReceiptWhen') }}</span>
          <select v-model="form.read_receipt_policy"
            class="px-2.5 py-2 border border-[var(--color-border)] rounded-md text-sm bg-[var(--color-bg)] text-[var(--color-text)] outline-none focus:border-teal w-auto">
            <option value="ask">{{ $t('settings.readReceiptAsk') }}</option>
            <option value="always">{{ $t('settings.readReceiptAlways') }}</option>
            <option value="never">{{ $t('settings.readReceiptNever') }}</option>
          </select>
        </label>

        <!-- Vacation autoresponder -->
        <div class="text-xs text-[var(--color-text-muted)] font-medium pt-1 border-t border-[var(--color-border)] mt-1">{{ $t('settings.vacationSection') }}</div>
        <div class="flex flex-col gap-2">
          <div class="flex items-center gap-2.5">
            <span class="text-sm text-[var(--color-text)] flex-1">{{ $t('settings.vacationEnable') }}</span>
            <label class="relative inline-flex items-center cursor-pointer">
              <input type="checkbox" v-model="form.vacation_enabled" class="sr-only peer" />
              <div class="w-9 h-5 bg-[var(--color-border)] peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-teal"></div>
            </label>
          </div>

          <template v-if="form.vacation_enabled">
            <label class="flex flex-col gap-1 text-xs text-[var(--color-text-muted)]">
              {{ $t('settings.vacationSubject') }}
              <input v-model="form.vacation_subject" type="text" :placeholder="$t('settings.vacationSubjectPlaceholder')"
                class="px-2.5 py-2 border border-[var(--color-border)] rounded-md text-sm bg-[var(--color-bg)] text-[var(--color-text)] outline-none focus:border-teal" />
            </label>
            <label class="flex flex-col gap-1 text-xs text-[var(--color-text-muted)]">
              {{ $t('settings.vacationMessage') }}
              <textarea v-model="form.vacation_body" :placeholder="$t('settings.vacationBodyPlaceholder')"
                class="px-2.5 py-2 border border-[var(--color-border)] rounded-md text-sm bg-[var(--color-bg)] text-[var(--color-text)] outline-none resize-y min-h-[80px] leading-relaxed focus:border-teal" />
            </label>
            <div class="flex gap-2">
              <label class="flex flex-col gap-1 text-xs text-[var(--color-text-muted)] flex-1">
                {{ $t('settings.vacationStartDate') }}
                <input v-model="form.vacation_start" type="date"
                  class="px-2.5 py-2 border border-[var(--color-border)] rounded-md text-sm bg-[var(--color-bg)] text-[var(--color-text)] outline-none focus:border-teal" />
              </label>
              <label class="flex flex-col gap-1 text-xs text-[var(--color-text-muted)] flex-1">
                {{ $t('settings.vacationEndDate') }}
                <input v-model="form.vacation_end" type="date"
                  class="px-2.5 py-2 border border-[var(--color-border)] rounded-md text-sm bg-[var(--color-bg)] text-[var(--color-text)] outline-none focus:border-teal" />
              </label>
            </div>
          </template>

          <!-- Status banner after save -->
          <div v-if="vacationStatus" :class="[
            'px-3 py-2 rounded-md text-xs',
            vacationStatus.type === 'active' && 'bg-teal/10 text-teal border border-teal/20',
            vacationStatus.type === 'warn' && 'bg-yellow-50 text-yellow-700 border border-yellow-200 dark:bg-yellow-900/20 dark:text-yellow-400 dark:border-yellow-800',
            vacationStatus.type === 'error' && 'bg-red-50 text-red-600 border border-red-200 dark:bg-red-900/20 dark:text-red-400 dark:border-red-800',
          ]">{{ vacationStatus.message }}</div>
        </div>

        <!-- Mail filters (only shown when ManageSieve is configured) -->
        <template v-if="settings.sieveConfigured">
          <div class="text-xs text-[var(--color-text-muted)] font-medium pt-1 border-t border-[var(--color-border)] mt-1">{{ $t('settings.mailFiltersSection') }}</div>
          <div class="flex items-center gap-2.5">
            <span class="text-sm text-[var(--color-text)] flex-1">{{ $t('settings.filtersDescription') }}</span>
            <button @click="showFilters = true"
              class="px-3 py-1.5 border border-[var(--color-border)] rounded-md bg-[var(--color-surface)] text-xs cursor-pointer text-[var(--color-text)] whitespace-nowrap hover:border-teal hover:text-teal">{{ $t('settings.manageFilters') }}</button>
          </div>
        </template>

        <div class="text-xs text-[var(--color-text-muted)] font-medium pt-1 border-t border-[var(--color-border)] mt-1">{{ $t('settings.securitySection') }}</div>
        <div class="flex flex-col gap-2">
          <div v-if="sessionsLoading" class="text-xs text-[var(--color-text-muted)]">{{ $t('settings.loadingSessions') }}</div>
          <div v-else-if="sessions.length" class="flex flex-col gap-1.5">
            <div v-for="s in sessions" :key="s.id"
              class="flex items-start gap-2 px-2.5 py-2 rounded-md bg-[var(--color-bg)] border border-[var(--color-border)]">
              <div class="flex-1 min-w-0">
                <div class="text-xs font-medium text-[var(--color-text)] truncate">{{ browserName(s.user_agent) }}<span v-if="s.current" class="ml-1.5 text-[10px] text-teal font-semibold">{{ $t('settings.thisDevice') }}</span></div>
                <div class="text-[10px] text-[var(--color-text-muted)] mt-0.5">
                  {{ $t('settings.signedIn', { date: formatDate(s.created_at) }) }} · {{ $t('settings.lastSeen', { date: formatDate(s.last_activity_at) }) }}
                </div>
              </div>
            </div>
          </div>
          <div class="flex gap-2 flex-wrap">
            <button @click="logoutOtherDevices" :disabled="logoutAllBusy || sessions.filter(s => !s.current).length === 0"
              class="px-3 py-1.5 border border-[var(--color-border)] rounded-md bg-[var(--color-surface)] text-xs cursor-pointer text-[var(--color-text)] whitespace-nowrap hover:border-red-500 hover:text-red-600 disabled:opacity-40 disabled:cursor-not-allowed">
              {{ $t('settings.logoutOtherDevices') }}
            </button>
            <button @click="logoutEverywhere" :disabled="logoutAllBusy"
              class="px-3 py-1.5 border border-[var(--color-border)] rounded-md bg-[var(--color-surface)] text-xs cursor-pointer text-[var(--color-text)] whitespace-nowrap hover:border-red-500 hover:text-red-600 disabled:opacity-40 disabled:cursor-not-allowed">
              {{ $t('settings.logoutEverywhere') }}
            </button>
          </div>
          <p v-if="logoutAllError" class="text-xs text-red-600">{{ logoutAllError }}</p>
        </div>

        <!-- 2FA -->
        <div class="text-xs text-[var(--color-text-muted)] font-semibold pt-1 mt-1">{{ $t('settings.twoFactorSection') }}</div>
        <div class="flex flex-col gap-2">
          <!-- Disabled state -->
          <template v-if="!settings.totpEnabled && twofa.step === 'idle'">
            <p class="text-xs text-[var(--color-text-muted)]">{{ $t('settings.twoFaDescription') }}</p>
            <div>
              <button @click="start2FASetup" :disabled="twofa.busy"
                class="px-3 py-1.5 border border-[var(--color-border)] rounded-md bg-[var(--color-surface)] text-xs cursor-pointer text-[var(--color-text)] whitespace-nowrap hover:border-teal hover:text-teal disabled:opacity-50">
                {{ twofa.busy ? $t('settings.loading2FA') : $t('settings.enable2FA') }}
              </button>
            </div>
          </template>

          <!-- Enrollment: show QR + verify -->
          <template v-else-if="twofa.step === 'enroll'">
            <p class="text-xs text-[var(--color-text-muted)]">{{ $t('settings.enrollPrompt') }}</p>
            <img v-if="twofa.qrPng" :src="'data:image/png;base64,' + twofa.qrPng" alt="QR code" class="w-40 h-40 rounded border border-[var(--color-border)]" />
            <p class="text-xs text-[var(--color-text-muted)]">{{ $t('settings.manualEntry') }} <span class="font-mono text-[var(--color-text)]">{{ twofa.secret }}</span></p>
            <div class="flex gap-2 items-center">
              <input v-model="twofa.code" type="text" inputmode="numeric" autocomplete="one-time-code"
                pattern="[0-9]{6}" maxlength="6" placeholder="000000"
                class="w-28 px-2.5 py-1.5 border border-[var(--color-border)] rounded-md text-sm outline-none bg-[var(--color-surface)] focus:border-teal font-mono tracking-widest text-center" />
              <button @click="confirm2FAEnroll" :disabled="twofa.busy || twofa.code.length !== 6"
                class="px-3 py-1.5 bg-teal text-white border-none rounded-md text-xs cursor-pointer disabled:opacity-50">
                {{ twofa.busy ? $t('settings.verifying') : $t('settings.verify') }}
              </button>
              <button @click="cancel2FA"
                class="px-3 py-1.5 border border-[var(--color-border)] rounded-md text-xs cursor-pointer text-[var(--color-text)] hover:border-[var(--color-text)]">
                {{ $t('settings.cancel') }}
              </button>
            </div>
            <p v-if="twofa.error" class="text-xs text-red-600">{{ twofa.error }}</p>
          </template>

          <!-- Recovery codes shown once after enrollment -->
          <template v-else-if="twofa.step === 'recovery'">
            <p class="text-xs font-medium text-[var(--color-text)]">{{ $t('settings.recoveryTitle') }}</p>
            <div class="grid grid-cols-2 gap-1 p-3 bg-[var(--color-bg)] border border-[var(--color-border)] rounded-md">
              <span v-for="c in twofa.recoveryCodes" :key="c" class="font-mono text-xs text-[var(--color-text)]">{{ c }}</span>
            </div>
            <button @click="twofa.step = 'idle'"
              class="self-start px-3 py-1.5 bg-teal text-white border-none rounded-md text-xs cursor-pointer">
              {{ $t('settings.done') }}
            </button>
          </template>

          <!-- Enabled state -->
          <template v-else-if="settings.totpEnabled && twofa.step === 'idle'">
            <p class="text-xs text-teal font-medium">{{ $t('settings.twoFaActive') }}</p>
            <div class="flex gap-2 flex-wrap">
              <button @click="twofa.step = 'regen-confirm'"
                class="px-3 py-1.5 border border-[var(--color-border)] rounded-md bg-[var(--color-surface)] text-xs cursor-pointer text-[var(--color-text)] whitespace-nowrap hover:border-teal hover:text-teal">
                {{ $t('settings.regenCodes') }}
              </button>
              <button @click="twofa.step = 'disable-confirm'"
                class="px-3 py-1.5 border border-[var(--color-border)] rounded-md bg-[var(--color-surface)] text-xs cursor-pointer whitespace-nowrap hover:border-red-500 hover:text-red-600">
                {{ $t('settings.disable2FA') }}
              </button>
            </div>
          </template>

          <!-- Confirm disable -->
          <template v-else-if="twofa.step === 'disable-confirm'">
            <p class="text-xs text-[var(--color-text-muted)]">{{ $t('settings.disablePrompt') }}</p>
            <div class="flex gap-2 items-center">
              <input v-model="twofa.code" type="text" inputmode="numeric" autocomplete="one-time-code"
                placeholder="000000" maxlength="8"
                class="w-28 px-2.5 py-1.5 border border-[var(--color-border)] rounded-md text-sm outline-none bg-[var(--color-surface)] focus:border-teal font-mono tracking-widest text-center" />
              <button @click="disable2FA" :disabled="twofa.busy || !twofa.code"
                class="px-3 py-1.5 bg-red-600 text-white border-none rounded-md text-xs cursor-pointer disabled:opacity-50">
                {{ twofa.busy ? $t('settings.disabling') : $t('settings.disable2FA') }}
              </button>
              <button @click="cancel2FA"
                class="px-3 py-1.5 border border-[var(--color-border)] rounded-md text-xs cursor-pointer text-[var(--color-text)] hover:border-[var(--color-text)]">
                {{ $t('settings.cancel') }}
              </button>
            </div>
            <p v-if="twofa.error" class="text-xs text-red-600">{{ twofa.error }}</p>
          </template>

          <!-- Confirm regen recovery codes -->
          <template v-else-if="twofa.step === 'regen-confirm'">
            <p class="text-xs text-[var(--color-text-muted)]">{{ $t('settings.regenPrompt') }}</p>
            <div class="flex gap-2 items-center">
              <input v-model="twofa.code" type="text" inputmode="numeric" autocomplete="one-time-code"
                placeholder="000000" maxlength="6"
                class="w-28 px-2.5 py-1.5 border border-[var(--color-border)] rounded-md text-sm outline-none bg-[var(--color-surface)] focus:border-teal font-mono tracking-widest text-center" />
              <button @click="regenRecoveryCodes" :disabled="twofa.busy || twofa.code.length !== 6"
                class="px-3 py-1.5 bg-teal text-white border-none rounded-md text-xs cursor-pointer disabled:opacity-50">
                {{ twofa.busy ? $t('settings.regenerating') : $t('settings.regenerate') }}
              </button>
              <button @click="cancel2FA"
                class="px-3 py-1.5 border border-[var(--color-border)] rounded-md text-xs cursor-pointer text-[var(--color-text)] hover:border-[var(--color-text)]">
                {{ $t('settings.cancel') }}
              </button>
            </div>
            <p v-if="twofa.error" class="text-xs text-red-600">{{ twofa.error }}</p>
          </template>
        </div>

        <!-- PGP -->
        <div class="text-xs text-[var(--color-text-muted)] font-medium pt-1 border-t border-[var(--color-border)] mt-1">{{ $t('settings.pgpSection') }}</div>
        <div class="flex flex-col gap-2">

          <!-- No key stored -->
          <template v-if="!pgp.hasKey">
            <p class="text-xs text-[var(--color-text-muted)]">{{ $t('settings.pgpNoKey') }}</p>
            <div class="flex gap-2 flex-wrap">
              <button @click="pgpMode = 'generate'"
                class="px-3 py-1.5 border border-[var(--color-border)] rounded-md bg-[var(--color-surface)] text-xs cursor-pointer hover:border-teal hover:text-teal">{{ $t('settings.generateKey') }}</button>
              <button @click="pgpMode = 'import'"
                class="px-3 py-1.5 border border-[var(--color-border)] rounded-md bg-[var(--color-surface)] text-xs cursor-pointer hover:border-teal hover:text-teal">{{ $t('settings.importKey') }}</button>
            </div>

            <!-- Generate form -->
            <div v-if="pgpMode === 'generate'" class="flex flex-col gap-2 p-3 bg-[var(--color-bg)] rounded-md border border-[var(--color-border)]">
              <p class="text-xs font-medium text-[var(--color-text)]">{{ $t('settings.generateTitle') }}</p>
              <input v-model="pgpForm.name" type="text" :placeholder="$t('settings.pgpNamePlaceholder')"
                class="px-2.5 py-1.5 border border-[var(--color-border)] rounded-md text-sm outline-none bg-[var(--color-surface)] focus:border-teal" />
              <input v-model="pgpForm.email" type="email" placeholder="your@email.com"
                class="px-2.5 py-1.5 border border-[var(--color-border)] rounded-md text-sm outline-none bg-[var(--color-surface)] focus:border-teal" />
              <input v-model="pgpForm.passphrase" type="password" :placeholder="$t('settings.passphraseProtect')"
                class="px-2.5 py-1.5 border border-[var(--color-border)] rounded-md text-sm outline-none bg-[var(--color-surface)] focus:border-teal" />
              <input v-model="pgpForm.passphrase2" type="password" :placeholder="$t('settings.confirmPassphrase')"
                class="px-2.5 py-1.5 border border-[var(--color-border)] rounded-md text-sm outline-none bg-[var(--color-surface)] focus:border-teal" />
              <p v-if="pgpError" class="text-xs text-red-600">{{ pgpError }}</p>
              <div class="flex gap-2">
                <button @click="generatePGPKey" :disabled="pgpBusy"
                  class="px-4 py-1.5 bg-teal text-white border-none rounded-md text-xs cursor-pointer disabled:opacity-60">{{ pgpBusy ? $t('settings.generating') : $t('settings.generate') }}</button>
                <button @click="pgpMode = null" class="px-3 py-1.5 border border-[var(--color-border)] rounded-md text-xs cursor-pointer">{{ $t('settings.cancel') }}</button>
              </div>
            </div>

            <!-- Import form -->
            <div v-if="pgpMode === 'import'" class="flex flex-col gap-2 p-3 bg-[var(--color-bg)] rounded-md border border-[var(--color-border)]">
              <p class="text-xs font-medium text-[var(--color-text)]">{{ $t('settings.importTitle') }}</p>
              <textarea v-model="pgpForm.armoredKey" rows="5" placeholder="-----BEGIN PGP PRIVATE KEY BLOCK-----"
                class="px-2.5 py-1.5 border border-[var(--color-border)] rounded-md text-xs font-mono outline-none bg-[var(--color-surface)] resize-y focus:border-teal" />
              <input v-model="pgpForm.passphrase" type="password" :placeholder="$t('settings.passphrasePlaceholder')"
                class="px-2.5 py-1.5 border border-[var(--color-border)] rounded-md text-sm outline-none bg-[var(--color-surface)] focus:border-teal" />
              <p v-if="pgpError" class="text-xs text-red-600">{{ pgpError }}</p>
              <div class="flex gap-2">
                <button @click="importPGPKey" :disabled="pgpBusy"
                  class="px-4 py-1.5 bg-teal text-white border-none rounded-md text-xs cursor-pointer disabled:opacity-60">{{ pgpBusy ? $t('settings.importing') : $t('settings.import') }}</button>
                <button @click="pgpMode = null" class="px-3 py-1.5 border border-[var(--color-border)] rounded-md text-xs cursor-pointer">{{ $t('settings.cancel') }}</button>
              </div>
            </div>
          </template>

          <!-- Key stored but locked -->
          <template v-else-if="pgp.isLocked">
            <p class="text-xs text-[var(--color-text-muted)]">{{ $t('settings.pgpLocked') }}</p>
            <div class="flex gap-2">
              <input v-model="pgpForm.passphrase" type="password" :placeholder="$t('settings.passphrasePlaceholder')"
                class="flex-1 px-2.5 py-1.5 border border-[var(--color-border)] rounded-md text-sm outline-none bg-[var(--color-surface)] focus:border-teal"
                @keydown.enter="unlockPGPKey" />
              <button @click="unlockPGPKey" :disabled="pgpBusy"
                class="px-4 py-1.5 bg-teal text-white border-none rounded-md text-xs cursor-pointer disabled:opacity-60">{{ pgpBusy ? $t('settings.unlocking') : $t('settings.unlock') }}</button>
            </div>
            <p v-if="pgpError" class="text-xs text-red-600">{{ pgpError }}</p>
            <button @click="confirmDeletePGPKey" class="self-start text-xs text-red-600 bg-none border-none cursor-pointer p-0 hover:underline">{{ $t('settings.deleteKeyLink') }}</button>
          </template>

          <!-- Key unlocked -->
          <template v-else-if="pgp.isUnlocked">
            <div class="px-3 py-2.5 bg-[var(--color-bg)] rounded-md border border-[var(--color-border)] flex flex-col gap-1">
              <p class="text-xs font-medium text-[var(--color-text)]">{{ $t('settings.keyUnlocked') }}</p>
              <p class="text-[11px] text-[var(--color-text-muted)] font-mono break-all">{{ pgp.fingerprint }}</p>
              <p v-if="pgp.userId" class="text-xs text-[var(--color-text-muted)]">{{ pgp.userId }}</p>
            </div>
            <div class="flex gap-2 flex-wrap">
              <button @click="exportPublicKey"
                class="px-3 py-1.5 border border-[var(--color-border)] rounded-md bg-[var(--color-surface)] text-xs cursor-pointer hover:border-teal hover:text-teal">{{ $t('settings.exportPublic') }}</button>
              <button @click="exportPrivateKey"
                class="px-3 py-1.5 border border-[var(--color-border)] rounded-md bg-[var(--color-surface)] text-xs cursor-pointer hover:border-teal hover:text-teal">{{ $t('settings.exportPrivate') }}</button>
              <button @click="pgp.lock()"
                class="px-3 py-1.5 border border-[var(--color-border)] rounded-md bg-[var(--color-surface)] text-xs cursor-pointer hover:bg-[var(--color-bg)]">{{ $t('settings.lock') }}</button>
              <button @click="confirmDeletePGPKey"
                class="px-3 py-1.5 border border-red-200 rounded-md bg-[var(--color-surface)] text-xs cursor-pointer text-red-600 hover:bg-[var(--color-bg)]">{{ $t('settings.deleteKey') }}</button>
            </div>
          </template>

        </div>

        <!-- Trusted image senders -->
        <div class="text-xs text-[var(--color-text-muted)] font-medium pt-1 border-t border-[var(--color-border)] mt-1">{{ $t('settings.trustedImageSection') }}</div>
        <div class="flex flex-col gap-1.5">
          <p v-if="settings.trustedImageSenders.length === 0" class="text-xs text-[var(--color-text-muted)]">{{ $t('settings.noTrustedSenders') }}</p>
          <div v-for="addr in settings.trustedImageSenders" :key="addr"
            class="flex items-center gap-2 px-2.5 py-1.5 rounded-md bg-[var(--color-bg)] border border-[var(--color-border)]">
            <span class="flex-1 text-xs font-mono text-[var(--color-text)] truncate">{{ addr }}</span>
            <button @click="revokeTrust(addr)"
              class="bg-none border-none text-xs cursor-pointer text-[var(--color-text-muted)] px-1 hover:text-red-600">{{ $t('settings.revoke') }}</button>
          </div>
        </div>

        <!-- Message templates -->
        <div class="text-xs text-[var(--color-text-muted)] font-medium pt-1 border-t border-[var(--color-border)] mt-1">{{ $t('settings.templatesSection') }}</div>
        <div class="flex items-center gap-2.5">
          <span class="text-sm text-[var(--color-text)] flex-1">{{ $t('settings.savedReplies') }}</span>
          <button @click="showTemplates = true"
            class="px-3 py-1.5 border border-[var(--color-border)] rounded-md bg-[var(--color-surface)] text-xs cursor-pointer text-[var(--color-text)] whitespace-nowrap hover:border-teal hover:text-teal">{{ $t('settings.manageTemplates') }}</button>
        </div>

        <div class="text-xs text-[var(--color-text-muted)] font-medium pt-1 border-t border-[var(--color-border)] mt-1">{{ $t('settings.identitiesSection') }}</div>
        <div class="flex flex-col gap-2">
          <div v-for="(id, i) in form.identities" :key="i" class="flex gap-1.5 items-center">
            <input v-model="id.name" type="text" :placeholder="$t('settings.namePlaceholder')"
              class="flex-1 px-2.5 py-2 border border-[var(--color-border)] rounded-md text-sm bg-[var(--color-bg)] text-[var(--color-text)] outline-none focus:border-teal" />
            <input v-model="id.email" type="email" placeholder="email@example.com"
              class="flex-[1.5] px-2.5 py-2 border border-[var(--color-border)] rounded-md text-sm bg-[var(--color-bg)] text-[var(--color-text)] outline-none focus:border-teal" />
            <button @click="removeIdentity(i)"
              class="bg-none border-none text-base cursor-pointer text-[var(--color-text-muted)] px-1.5 py-1 shrink-0 rounded hover:bg-[var(--color-teal-light)]">×</button>
          </div>
          <button @click="addIdentity"
            class="bg-none border border-dashed border-[var(--color-border)] rounded-md px-3 py-1.5 text-xs cursor-pointer text-[var(--color-text-muted)] text-left hover:border-teal hover:text-teal">{{ $t('settings.addIdentity') }}</button>
        </div>

        <!-- Language -->
        <div class="text-xs text-[var(--color-text-muted)] font-medium pt-1 border-t border-[var(--color-border)] mt-1">{{ $t('settings.language') }}</div>
        <select v-model="form.locale"
          class="px-2.5 py-2 border border-[var(--color-border)] rounded-md text-sm bg-[var(--color-bg)] text-[var(--color-text)] outline-none focus:border-teal w-auto">
          <option value="en">{{ $t('settings.languageEn') }}</option>
        </select>
      </div>

      <!-- Footer -->
      <div class="px-4 py-3 border-t border-[var(--color-border)] flex items-center gap-4 sticky bottom-0 bg-[var(--color-surface)]">
        <button @click="save" :disabled="saving"
          class="px-5 py-1.5 bg-teal text-white border-none rounded-md text-sm font-medium cursor-pointer disabled:opacity-60 disabled:cursor-not-allowed">
          {{ saving ? $t('settings.saving') : saved ? $t('settings.saved') : $t('settings.save') }}
        </button>
        <p v-if="error" class="text-xs text-red-600">{{ error }}</p>
      </div>
    </div>
  </div>
  <FiltersModal :visible="showFilters" @close="showFilters = false" />
  <TemplatesModal :visible="showTemplates" @close="showTemplates = false" />
</template>

<script setup>
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSettingsStore } from '../stores/settings'
import { useAuthStore } from '../stores/auth'
import { usePGPStore } from '../stores/pgp'
import { apiFetch } from '../api'
import FiltersModal from './FiltersModal.vue'
import TemplatesModal from './TemplatesModal.vue'

const { t } = useI18n()
const settings = useSettingsStore()
const auth = useAuthStore()
const pgp = usePGPStore()
const visible = ref(false)
const saving = ref(false)
const saved = ref(false)
const error = ref('')
const showFilters = ref(false)
const showTemplates = ref(false)
const notifPermission = ref(typeof Notification !== 'undefined' ? Notification.permission : 'denied')

const sessions = ref([])
const sessionsLoading = ref(false)
const logoutAllBusy = ref(false)
const logoutAllError = ref('')

// 2FA state
const twofa = reactive({
  step: 'idle',  // 'idle' | 'enroll' | 'recovery' | 'disable-confirm' | 'regen-confirm'
  busy: false,
  error: '',
  secret: '',
  qrPng: '',
  code: '',
  recoveryCodes: [],
})

async function start2FASetup() {
  twofa.busy = true
  twofa.error = ''
  try {
    const res = await apiFetch('/api/2fa/setup')
    if (!res.ok) throw new Error('Setup failed')
    const data = await res.json()
    twofa.secret = data.secret
    twofa.qrPng = data.qr_png_b64
    twofa.code = ''
    twofa.step = 'enroll'
  } catch (e) {
    twofa.error = e.message
  } finally {
    twofa.busy = false
  }
}

async function confirm2FAEnroll() {
  twofa.busy = true
  twofa.error = ''
  try {
    const res = await apiFetch('/api/2fa/enable', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ code: twofa.code }),
    })
    if (!res.ok) {
      const d = await res.json().catch(() => ({}))
      throw new Error(d.error || 'Invalid code')
    }
    const data = await res.json()
    twofa.recoveryCodes = data.recovery_codes
    twofa.step = 'recovery'
    settings.settings.totp_enabled = true
  } catch (e) {
    twofa.error = e.message
    twofa.code = ''
  } finally {
    twofa.busy = false
  }
}

async function disable2FA() {
  twofa.busy = true
  twofa.error = ''
  try {
    const res = await apiFetch('/api/2fa/disable', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ code: twofa.code }),
    })
    if (!res.ok) {
      const d = await res.json().catch(() => ({}))
      throw new Error(d.error || 'Invalid code')
    }
    settings.settings.totp_enabled = false
    twofa.step = 'idle'
    twofa.code = ''
  } catch (e) {
    twofa.error = e.message
    twofa.code = ''
  } finally {
    twofa.busy = false
  }
}

async function regenRecoveryCodes() {
  twofa.busy = true
  twofa.error = ''
  try {
    const res = await apiFetch('/api/2fa/recovery-codes', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ code: twofa.code }),
    })
    if (!res.ok) {
      const d = await res.json().catch(() => ({}))
      throw new Error(d.error || 'Invalid code')
    }
    const data = await res.json()
    twofa.recoveryCodes = data.recovery_codes
    twofa.step = 'recovery'
    twofa.code = ''
  } catch (e) {
    twofa.error = e.message
    twofa.code = ''
  } finally {
    twofa.busy = false
  }
}

function cancel2FA() {
  twofa.step = 'idle'
  twofa.code = ''
  twofa.error = ''
}

// PGP state
const pgpMode = ref(null)  // null | 'generate' | 'import'
const pgpBusy = ref(false)
const pgpError = ref('')
const pgpForm = reactive({ name: '', email: '', passphrase: '', passphrase2: '', armoredKey: '' })

const form = reactive({ display_name: '', signature: '', identities: [], poll_interval: 120, undo_send_delay: 0, calendar_reminder_minutes: 30, read_receipt_policy: 'ask', vacation_enabled: false, vacation_subject: '', vacation_body: '', vacation_start: '', vacation_end: '', locale: 'en' })
const vacationStatus = ref(null) // { type: 'active'|'warn'|'error', message: string }

async function open() {
  if (!settings.loaded) await settings.fetchSettings()
  form.display_name = settings.settings.display_name ?? ''
  form.signature = settings.settings.signature ?? ''
  form.identities = settings.identities.map(id => ({ ...id }))
  form.poll_interval = settings.pollInterval
  form.undo_send_delay = settings.undoSendDelay
  form.calendar_reminder_minutes = settings.reminderMinutes
  form.read_receipt_policy = settings.readReceiptPolicy
  form.locale = settings.locale
  notifPermission.value = typeof Notification !== 'undefined' ? Notification.permission : 'denied'
  saved.value = false
  error.value = ''
  logoutAllError.value = ''
  vacationStatus.value = null
  twofa.step = 'idle'
  twofa.code = ''
  twofa.error = ''
  pgpMode.value = null
  pgpError.value = ''
  pgpForm.passphrase = ''
  pgpForm.passphrase2 = ''
  // Pre-fill generate form with account info
  pgpForm.name = settings.settings.display_name ?? ''
  pgpForm.email = auth.user?.username ?? ''
  // Load vacation state from server
  fetchVacation()
  visible.value = true
  fetchSessions()
  pgp.fetchKey()
}

async function fetchVacation() {
  try {
    const res = await apiFetch('/api/vacation')
    if (!res.ok) return
    const data = await res.json()
    form.vacation_enabled = data.enabled
    form.vacation_subject = data.subject ?? ''
    form.vacation_body = data.body ?? ''
    form.vacation_start = data.start ?? ''
    form.vacation_end = data.end ?? ''
    if (data.enabled) {
      vacationStatus.value = data.sieve_active
        ? { type: 'active', message: t('settings.vacationActiveSieve') }
        : data.sieve_configured
          ? { type: 'warn', message: t('settings.vacationWarnSieve') }
          : null
    }
  } catch {
    // Non-critical — silently ignore
  }
}

async function fetchSessions() {
  sessionsLoading.value = true
  try {
    const res = await apiFetch('/api/auth/sessions')
    if (res.ok) sessions.value = await res.json()
  } finally {
    sessionsLoading.value = false
  }
}

async function logoutOtherDevices() {
  logoutAllBusy.value = true
  logoutAllError.value = ''
  try {
    const res = await apiFetch('/api/auth/sessions', {
      method: 'DELETE',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ include_current: false }),
    })
    if (!res.ok) throw new Error()
    await fetchSessions()
  } catch {
    logoutAllError.value = t('settings.couldNotLogout')
  } finally {
    logoutAllBusy.value = false
  }
}

async function logoutEverywhere() {
  logoutAllBusy.value = true
  logoutAllError.value = ''
  try {
    const res = await apiFetch('/api/auth/sessions', {
      method: 'DELETE',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ include_current: true }),
    })
    if (!res.ok) throw new Error()
    await auth.logout()
  } catch {
    logoutAllError.value = t('settings.couldNotLogoutEverywhere')
    logoutAllBusy.value = false
  }
}

function browserName(ua) {
  if (!ua) return 'Unknown browser'
  if (/Edg\//.test(ua)) return 'Microsoft Edge'
  if (/Firefox\//.test(ua)) return 'Firefox'
  if (/Chrome\//.test(ua)) return 'Chrome'
  if (/Safari\//.test(ua)) return 'Safari'
  return ua.length > 60 ? ua.slice(0, 60) + '…' : ua
}

function formatDate(iso) {
  if (!iso) return 'never'
  const d = new Date(iso)
  const now = new Date()
  const diffMs = now - d
  const diffMin = Math.floor(diffMs / 60000)
  if (diffMin < 1) return 'just now'
  if (diffMin < 60) return `${diffMin}m ago`
  const diffH = Math.floor(diffMin / 60)
  if (diffH < 24) return `${diffH}h ago`
  const diffD = Math.floor(diffH / 24)
  if (diffD < 7) return `${diffD}d ago`
  return d.toLocaleDateString()
}

function close() {
  visible.value = false
}

async function enableNotifications() {
  const result = await Notification.requestPermission()
  notifPermission.value = result
  if (result === 'granted') {
    await settings.saveSettings({ notifications_enabled: 'true' })
  }
}

async function disableNotifications() {
  await settings.saveSettings({ notifications_enabled: 'false' })
}

async function revokeTrust(email) {
  await settings.untrustImageSender(email)
}

function addIdentity() {
  form.identities.push({ name: '', email: '' })
}

function removeIdentity(i) {
  form.identities.splice(i, 1)
}

async function save() {
  saving.value = true
  error.value = ''
  try {
    const validIdentities = form.identities.filter(id => id.email.trim())
    await settings.saveSettings({
      display_name: form.display_name,
      signature: form.signature,
      identities: JSON.stringify(validIdentities),
      poll_interval: String(form.poll_interval),
      undo_send_delay: String(form.undo_send_delay),
      calendar_reminder_minutes: String(form.calendar_reminder_minutes),
      read_receipt_policy: form.read_receipt_policy,
      locale: form.locale,
    })
    // Save vacation settings separately (needs its own endpoint for Sieve side-effects).
    const vacResult = await settings.saveVacation({
      enabled: form.vacation_enabled,
      subject: form.vacation_subject,
      body: form.vacation_body,
      start: form.vacation_start,
      end: form.vacation_end,
    })
    if (form.vacation_enabled) {
      if (vacResult.sieve_active) {
        vacationStatus.value = { type: 'active', message: t('settings.vacationActiveSieve') }
      } else if (vacResult.sieve_configured) {
        vacationStatus.value = { type: vacResult.sieve_error ? 'error' : 'warn',
          message: vacResult.sieve_error ? 'Server error: ' + vacResult.sieve_error : t('settings.vacationWarnSieve') }
      } else {
        vacationStatus.value = null
      }
    } else {
      vacationStatus.value = null
    }
    saved.value = true
    setTimeout(() => { saved.value = false }, 2000)
  } catch (e) {
    error.value = e.message || t('settings.couldNotSave')
  } finally {
    saving.value = false
  }
}

function onKeydown(e) { if (e.key === 'Escape' && visible.value) close() }
onMounted(() => document.addEventListener('keydown', onKeydown))
onUnmounted(() => document.removeEventListener('keydown', onKeydown))

// ── PGP helpers ───────────────────────────────────────────────────────────

async function generatePGPKey() {
  pgpError.value = ''
  if (!pgpForm.name.trim() || !pgpForm.email.trim()) { pgpError.value = t('settings.pgpNameAndEmailRequired'); return }
  if (!pgpForm.passphrase) { pgpError.value = t('settings.pgpPassphraseRequired'); return }
  if (pgpForm.passphrase !== pgpForm.passphrase2) { pgpError.value = t('settings.pgpPassphraseMismatch'); return }
  pgpBusy.value = true
  try {
    await pgp.generateKey(pgpForm.name.trim(), pgpForm.email.trim(), pgpForm.passphrase)
    pgpMode.value = null
    pgpForm.passphrase = ''
    pgpForm.passphrase2 = ''
  } catch (e) {
    pgpError.value = e.message || t('settings.pgpKeyGenerationFailed')
  } finally {
    pgpBusy.value = false
  }
}

async function importPGPKey() {
  pgpError.value = ''
  if (!pgpForm.armoredKey.trim()) { pgpError.value = t('settings.pgpPasteKey'); return }
  if (!pgpForm.passphrase) { pgpError.value = t('settings.pgpPassphraseRequired'); return }
  pgpBusy.value = true
  try {
    await pgp.importKey(pgpForm.armoredKey.trim(), pgpForm.passphrase)
    pgpMode.value = null
    pgpForm.armoredKey = ''
    pgpForm.passphrase = ''
  } catch (e) {
    pgpError.value = e.message || t('settings.pgpImportFailed')
  } finally {
    pgpBusy.value = false
  }
}

async function unlockPGPKey() {
  pgpError.value = ''
  if (!pgpForm.passphrase) { pgpError.value = t('settings.pgpEnterPassphrase'); return }
  pgpBusy.value = true
  try {
    await pgp.unlock(pgpForm.passphrase)
    pgpForm.passphrase = ''
  } catch {
    pgpError.value = t('settings.pgpWrongPassphrase')
  } finally {
    pgpBusy.value = false
  }
}

async function confirmDeletePGPKey() {
  if (!confirm(t('settings.deleteKeyConfirm'))) return
  await pgp.deleteKey()
}

function downloadText(text, filename, mime = 'text/plain') {
  const a = document.createElement('a')
  a.href = URL.createObjectURL(new Blob([text], { type: mime }))
  a.download = filename
  a.click()
  URL.revokeObjectURL(a.href)
}

function exportPublicKey() {
  const armored = pgp.armoredPublicKey()
  if (armored) downloadText(armored, 'publickey.asc', 'application/pgp-keys')
}

function exportPrivateKey() {
  if (!pgp.encryptedKey) return
  downloadText(pgp.encryptedKey, 'privatekey.asc', 'application/pgp-keys')
}

defineExpose({ open, close })
</script>
