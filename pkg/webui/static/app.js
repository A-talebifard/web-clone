// WebClone GUI — Front-end Logic
//
// The UI is organized around a JOB QUEUE. The user builds a list of jobs on
// the Jobs tab (each job = one site to mirror, with its own URL, output
// folder, and full crawler options). When the user clicks "Start All", the
// entire list is POSTed to /api/jobs/start, which runs every job
// sequentially on the backend. SSE events stream back with a job_index
// field so we can highlight the active job in the list and show "Job 2 of 5"
// in the progress tab.

// Translations
const translations = {
    fa: {
        "app.title": "وب‌کلون",
        "app.subtitle": "دانلود کامل وب‌سایت‌ها",
        "app.language_toggle": "EN",
        "nav.jobs": "صف دانلود",
        "nav.controls": "کنترل",
        "nav.progress": "پیشرفت",
        "nav.logs": "لاگ‌ها",
        "nav.about": "درباره",
        "jobs.title": "صف دانلود",
        "jobs.add": "افزودن شغل",
        "jobs.start_all": "شروع همه",
        "jobs.clear_all": "پاک کردن همه",
        "jobs.empty": "هنوز شغلی اضافه نشده. روی «افزودن شغل» بزنید تا یک سایت جدید پیکربندی کنید.",
        "jobs.modal.add_title": "افزودن شغل جدید",
        "jobs.modal.edit_title": "ویرایش شغل",
        "jobs.modal.advanced": "تنظیمات پیشرفته",
        "jobs.modal.cancel": "انصراف",
        "jobs.modal.save": "ذخیره شغل",
        "jobs.edit": "ویرایش",
        "jobs.delete": "حذف",
        "jobs.status.pending": "در انتظار",
        "jobs.status.running": "در حال اجرا",
        "jobs.status.paused": "متوقف موقت",
        "jobs.status.done": "تکمیل شد",
        "jobs.status.failed": "خطا",
        "jobs.meta.workers": "همزمانی:",
        "jobs.meta.max_urls": "حداکثر URL:",
        "jobs.meta.same_domain": "هم‌دامنه",
        "jobs.meta.cross_domain": "بین‌دامنه",
        "jobs.confirm_clear": "همه شغل‌ها حذف شوند؟",
        "jobs.confirm_delete": "این شغل حذف شود؟",
        "jobs.need_jobs": "حداقل یک شغل اضافه کنید",
        "jobs.need_url": "URL را وارد کنید",
        "jobs.need_output": "پوشه خروجی را وارد کنید",
        "settings.url_label": "آدرس سایت (URL)",
        "settings.url_placeholder": "https://example.com",
        "settings.output_label": "پوشه خروجی",
        "settings.output_placeholder": "مسیر ذخیره فایل‌ها",
        "settings.output_browse": "انتخاب...",
        "settings.workers_label": "تعداد همزمانی",
        "settings.max_urls_label": "حداکثر تعداد URL",
        "settings.max_depth_label": "حداکثر عمق",
        "settings.timeout_label": "انقضای درخواست (ثانیه)",
        "flags.same_domain": "فقط هم‌دامنه",
        "flags.allow_subdomains": "ساب‌دامین‌ها",
        "flags.skip_assets": "فقط HTML",
        "flags.insecure_tls": "نادیده گرفتن TLS",
        "flags.manifest": "manifest.json",
        "controls.title": "کنترل خزش",
        "controls.pause": "مکث",
        "controls.resume": "ادامه",
        "controls.stop": "توقف",
        "controls.open_output": "باز کردن پوشه",
        "controls.serve": "نمایش در مرورگر",
        "controls.status_idle": "آماده",
        "controls.status_running": "در حال خزش...",
        "controls.status_paused": "متوقف موقت",
        "controls.status_done": "تکمیل شد",
        "controls.status_failed": "خطا",
        "progress.title": "پیشرفت زنده",
        "progress.current_job": "شغل فعلی",
        "progress.pages": "صفحات HTML",
        "progress.assets": "فایل‌ها",
        "progress.downloaded": "دانلود شده",
        "progress.failed": "خطاها",
        "progress.skipped": "رد شده",
        "progress.elapsed": "زمان",
        "progress.current_url": "URL فعلی",
        "progress.last_saved": "آخرین فایل",
        "progress.speed": "سرعت دانلود",
        "progress.avg_speed": "میانگین",
        "progress.peak_speed": "بیشینه",
        "advanced.proxy_label": "پراکسی",
        "advanced.proxy_placeholder": "http://localhost:8080",
        "advanced.user_agent_label": "User-Agent سفارشی",
        "advanced.cookies_label": "کوکی‌ها",
        "advanced.allowed_hosts_label": "میزبان‌های مجاز",
        "advanced.asset_ext_label": "پسوندهای asset",
        "logs.title": "لاگ‌ها",
        "logs.clear": "پاک کردن",
        "logs.copy": "کپی",
        "logs.filter_placeholder": "فیلتر (مثلاً error)",
        "about.title": "درباره وب‌کلون",
        "about.version": "نسخه",
        "about.description": "وب‌کلون ابزاری است برای دانلود کامل وب‌سایت‌ها به‌صورت آفلاین، با حفظ ساختار URL و بازنویسی لینک‌ها برای کار کردن آفلاین. از فایل‌های صوتی، تصویری، زیرنویس و تمام assetها پشتیبانی می‌کند.",
        "about.license": "لایسنس",
        "about.github": "مشاهده در گیت‌هاب",
        "about.powered_by": "ساخته‌شده با",
        "about.developer": "توسعه‌دهنده: a-talebifard",
        "about.footer": "برای مشاهده در گیت‌هاب کلیک کنید",
        "error.crawl_failed": "خطا در خزش",
        "toast.started": "صف دانلود شروع شد",
        "toast.stopped": "خزش متوقف شد",
        "toast.paused": "خزش متوقف موقت شد",
        "toast.resumed": "خزش از سر گرفته شد",
        "toast.completed": "خزش تکمیل شد",
        "toast.copied": "لاگ‌ها کپی شد",
        "toast.job_added": "شغل اضافه شد",
        "toast.job_updated": "شغل به‌روزرسانی شد",
        "toast.job_deleted": "شغل حذف شد",
        "toast.error": "خطا"
    },
    en: {
        "app.title": "WebClone",
        "app.subtitle": "Download entire websites",
        "app.language_toggle": "FA",
        "nav.jobs": "Job Queue",
        "nav.controls": "Controls",
        "nav.progress": "Progress",
        "nav.logs": "Logs",
        "nav.about": "About",
        "jobs.title": "Download Queue",
        "jobs.add": "Add Job",
        "jobs.start_all": "Start All",
        "jobs.clear_all": "Clear All",
        "jobs.empty": "No jobs yet. Click \"Add Job\" to configure a new site.",
        "jobs.modal.add_title": "Add New Job",
        "jobs.modal.edit_title": "Edit Job",
        "jobs.modal.advanced": "Advanced Settings",
        "jobs.modal.cancel": "Cancel",
        "jobs.modal.save": "Save Job",
        "jobs.edit": "Edit",
        "jobs.delete": "Delete",
        "jobs.status.pending": "Pending",
        "jobs.status.running": "Running",
        "jobs.status.paused": "Paused",
        "jobs.status.done": "Done",
        "jobs.status.failed": "Failed",
        "jobs.meta.workers": "Workers:",
        "jobs.meta.max_urls": "Max URLs:",
        "jobs.meta.same_domain": "Same-domain",
        "jobs.meta.cross_domain": "Cross-domain",
        "jobs.confirm_clear": "Delete all jobs?",
        "jobs.confirm_delete": "Delete this job?",
        "jobs.need_jobs": "Add at least one job",
        "jobs.need_url": "Please enter a URL",
        "jobs.need_output": "Please enter an output directory",
        "settings.url_label": "Website URL",
        "settings.url_placeholder": "https://example.com",
        "settings.output_label": "Output Directory",
        "settings.output_placeholder": "Path to save files",
        "settings.output_browse": "Browse...",
        "settings.workers_label": "Concurrent Workers",
        "settings.max_urls_label": "Max URLs",
        "settings.max_depth_label": "Max Depth",
        "settings.timeout_label": "Timeout (seconds)",
        "flags.same_domain": "Same-domain only",
        "flags.allow_subdomains": "Allow subdomains",
        "flags.skip_assets": "HTML only",
        "flags.insecure_tls": "Skip TLS",
        "flags.manifest": "manifest.json",
        "controls.title": "Crawl Controls",
        "controls.pause": "Pause",
        "controls.resume": "Resume",
        "controls.stop": "Stop",
        "controls.open_output": "Open Folder",
        "controls.serve": "Open in Browser",
        "controls.status_idle": "Idle",
        "controls.status_running": "Crawling...",
        "controls.status_paused": "Paused",
        "controls.status_done": "Completed",
        "controls.status_failed": "Failed",
        "progress.title": "Live Progress",
        "progress.current_job": "Current Job",
        "progress.pages": "HTML Pages",
        "progress.assets": "Assets",
        "progress.downloaded": "Downloaded",
        "progress.failed": "Errors",
        "progress.skipped": "Skipped",
        "progress.elapsed": "Elapsed",
        "progress.current_url": "Current URL",
        "progress.last_saved": "Last saved",
        "progress.speed": "Download speed",
        "progress.avg_speed": "Average",
        "progress.peak_speed": "Peak",
        "advanced.proxy_label": "Proxy",
        "advanced.proxy_placeholder": "http://localhost:8080",
        "advanced.user_agent_label": "Custom User-Agent",
        "advanced.cookies_label": "Cookies",
        "advanced.allowed_hosts_label": "Allowed hosts",
        "advanced.asset_ext_label": "Asset extensions",
        "logs.title": "Logs",
        "logs.clear": "Clear",
        "logs.copy": "Copy",
        "logs.filter_placeholder": "Filter (e.g. error)",
        "about.title": "About WebClone",
        "about.version": "Version",
        "about.description": "WebClone is a tool for downloading entire websites for offline use, preserving URL structure and rewriting links so the mirror works offline. Supports audio, video, subtitle tracks, and all asset types.",
        "about.license": "License",
        "about.github": "View on GitHub",
        "about.powered_by": "Built with",
        "about.developer": "Developer: A-talebifard",
        "about.footer": "Click to view on GitHub",
        "error.crawl_failed": "Crawl failed",
        "toast.started": "Download queue started",
        "toast.stopped": "Crawl stopped",
        "toast.paused": "Crawl paused",
        "toast.resumed": "Crawl resumed",
        "toast.completed": "Crawl completed",
        "toast.copied": "Logs copied",
        "toast.job_added": "Job added",
        "toast.job_updated": "Job updated",
        "toast.job_deleted": "Job deleted",
        "toast.error": "Error"
    }
};

let currentLang = 'fa';

// ===== State =====
//
// jobs is the user's saved list of crawl jobs. Each job has its own URL,
// output dir, and crawler options. Persisted to localStorage so it survives
// page refreshes.
let jobs = [];

// editingJobIndex is -1 when adding a new job, otherwise the 0-based index
// of the job being edited in the modal.
let editingJobIndex = -1;

// Queue run state. currentJobIndex mirrors the backend's job_index field
// so we can highlight the active job card. -1 means no job is running.
let currentJobIndex = -1;
let totalJobs = 0;
let isPaused = false;
let isRunning = false;

// SSE / timers
let eventSource = null;
let allLogs = [];
let startTime = null;
let elapsedTimer = null;

// Speed chart state — sample bytes-downloaded every SPEED_SAMPLE_MS and
// keep the last SPEED_MAX_SAMPLES samples for the rolling line chart.
const SPEED_SAMPLE_MS = 1000;
const SPEED_MAX_SAMPLES = 120;
let speedSamples = [];
let lastSampleBytes = 0;
let lastSampleTime = 0;
let peakSpeed = 0;
let speedTimer = null;
let canvasCtx = null;

// ===== Initialize =====
document.addEventListener('DOMContentLoaded', () => {
    setupNavigation();
    setupLanguageToggle();
    setupThemeToggle();
    setupJobsTab();
    setupControls();
    setupLogs();
    setupModal();
    setupSpeedChart();
    applyTranslations();
    loadTheme();
    loadJobs();
    renderJobs();
    pollStatus();
});

// ===== Navigation =====
function setupNavigation() {
    const navItems = document.querySelectorAll('.nav-item');
    const panels = document.querySelectorAll('.tab-panel');
    navItems.forEach(item => {
        item.addEventListener('click', () => {
            navItems.forEach(n => n.classList.remove('active'));
            panels.forEach(p => p.classList.remove('active'));
            item.classList.add('active');
            const tabId = item.getAttribute('data-tab');
            document.getElementById('tab-' + tabId).classList.add('active');
        });
    });
}

// ===== Language =====
function setupLanguageToggle() {
    const btn = document.getElementById('langToggle');
    btn.addEventListener('click', () => {
        currentLang = currentLang === 'fa' ? 'en' : 'fa';
        localStorage.setItem('webclone-lang', currentLang);
        applyTranslations();
        renderJobs();
    });
}

function t(key) {
    return (translations[currentLang] && translations[currentLang][key]) || key;
}

function applyTranslations() {
    document.documentElement.lang = currentLang;
    document.documentElement.dir = currentLang === 'fa' ? 'rtl' : 'ltr';
    document.querySelectorAll('[data-i18n]').forEach(el => {
        const key = el.getAttribute('data-i18n');
        el.textContent = t(key);
    });
    document.querySelectorAll('[data-i18n-placeholder]').forEach(el => {
        const key = el.getAttribute('data-i18n-placeholder');
        el.placeholder = t(key);
    });
    const urlInput = document.getElementById('modal_url');
    if (urlInput && !urlInput.value) urlInput.placeholder = t('settings.url_placeholder');
    const outputInput = document.getElementById('outputDir');
    if (outputInput && !outputInput.value) outputInput.placeholder = t('settings.output_placeholder');
    document.getElementById('langToggle').textContent = t('app.language_toggle');
    updatePauseButtonLabel();
}

// ===== Theme =====
function setupThemeToggle() {
    const btn = document.getElementById('themeToggle');
    if (!btn) return;
    btn.addEventListener('click', () => {
        const current = document.documentElement.getAttribute('data-theme');
        const next = current === 'light' ? 'dark' : 'light';
        applyTheme(next);
        localStorage.setItem('webclone-theme', next);
    });
}

function applyTheme(theme) {
    if (theme === 'light') {
        document.documentElement.setAttribute('data-theme', 'light');
    } else {
        document.documentElement.removeAttribute('data-theme');
    }
    const moon = document.querySelector('.icon-moon');
    const sun = document.querySelector('.icon-sun');
    if (moon && sun) {
        moon.style.display = theme === 'light' ? 'none' : 'block';
        sun.style.display = theme === 'light' ? 'block' : 'none';
    }
    // Redraw the speed chart so its colors match the new theme.
    drawSpeedChart();
}

function loadTheme() {
    const saved = localStorage.getItem('webclone-theme');
    applyTheme(saved === 'light' ? 'light' : 'dark');
}

// Restore saved language on load
(function restoreLang() {
    const saved = localStorage.getItem('webclone-lang');
    if (saved === 'en' || saved === 'fa') {
        currentLang = saved;
    }
})();

// ===== Jobs Tab =====
function setupJobsTab() {
    document.getElementById('addJobBtn').addEventListener('click', () => openModal(-1));
    document.getElementById('startAllBtn').addEventListener('click', startAllJobs);
    document.getElementById('clearJobsBtn').addEventListener('click', clearAllJobs);
    document.getElementById('browseBtn').addEventListener('click', () => pickFolderInto('outputDir'));
}

function loadJobs() {
    const saved = localStorage.getItem('webclone-jobs');
    if (!saved) return;
    try {
        jobs = JSON.parse(saved);
        // Reset any leftover run state from a previous session; the actual
        // queue state is recovered separately via pollStatus().
        for (const j of jobs) {
            j.status = 'pending';
        }
    } catch (e) {
        jobs = [];
    }
}

function saveJobs() {
    localStorage.setItem('webclone-jobs', JSON.stringify(jobs));
}

function renderJobs() {
    const list = document.getElementById('jobsList');
    const empty = document.getElementById('jobsEmpty');
    list.innerHTML = '';
    if (jobs.length === 0) {
        empty.style.display = 'block';
        return;
    }
    empty.style.display = 'none';
    jobs.forEach((job, i) => {
        list.appendChild(renderJobCard(job, i));
    });
}

function renderJobCard(job, index) {
    const card = document.createElement('div');
    // Determine the card's visual state. While the queue is running we use
    // currentJobIndex (synced from backend events); otherwise we fall back
    // to the job's stored status from a previous run.
    let stateClass = '';
    let statusKey = 'jobs.status.pending';
    if (isRunning && index === currentJobIndex) {
        if (isPaused) {
            stateClass = 'paused';
            statusKey = 'jobs.status.paused';
        } else {
            stateClass = 'running';
            statusKey = 'jobs.status.running';
        }
    } else if (isRunning && index < currentJobIndex) {
        stateClass = 'done';
        statusKey = 'jobs.status.done';
    } else if (job.status === 'done') {
        stateClass = 'done';
        statusKey = 'jobs.status.done';
    } else if (job.status === 'failed') {
        stateClass = 'failed';
        statusKey = 'jobs.status.failed';
    }
    card.className = 'job-card ' + stateClass;
    card.dataset.index = index;

    const domainLabel = job.same_domain ? t('jobs.meta.same_domain') : t('jobs.meta.cross_domain');
    const outputDisplay = job.output_dir || '—';

    card.innerHTML = `
        <div class="job-index">${index + 1}</div>
        <div class="job-info">
            <div class="job-url">${escapeHtml(job.url)}</div>
            <div class="job-meta">
                <span class="meta-output">📁 ${escapeHtml(outputDisplay)}</span>
                <span>${t('jobs.meta.workers')} ${job.workers || 5}</span>
                <span>${t('jobs.meta.max_urls')} ${job.max_urls || 10000}</span>
                <span>${domainLabel}</span>
            </div>
        </div>
        <span class="job-status ${stateClass || 'pending'}">${t(statusKey)}</span>
        <div class="job-actions">
            <button class="btn btn-secondary" data-action="edit" data-index="${index}">${t('jobs.edit')}</button>
            <button class="btn btn-danger" data-action="delete" data-index="${index}">${t('jobs.delete')}</button>
        </div>
    `;
    // Wire up the action buttons. We attach handlers here rather than using
    // inline onclick so we can pass the index safely without string coercion.
    card.querySelector('[data-action="edit"]').addEventListener('click', () => openModal(index));
    card.querySelector('[data-action="delete"]').addEventListener('click', () => deleteJob(index));
    return card;
}

function clearAllJobs() {
    if (isRunning) {
        showToast(t('toast.error'), 'error');
        return;
    }
    if (!confirm(t('jobs.confirm_clear'))) return;
    jobs = [];
    saveJobs();
    renderJobs();
}

function deleteJob(index) {
    if (isRunning) {
        showToast(t('toast.error'), 'error');
        return;
    }
    if (!confirm(t('jobs.confirm_delete'))) return;
    jobs.splice(index, 1);
    saveJobs();
    renderJobs();
    showToast(t('toast.job_deleted'), 'info');
}

// ===== Modal =====
function setupModal() {
    document.getElementById('modalCloseBtn').addEventListener('click', closeModal);
    document.getElementById('modalCancelBtn').addEventListener('click', closeModal);
    document.getElementById('modalSaveBtn').addEventListener('click', saveJobFromModal);
    document.getElementById('modal_browseBtn').addEventListener('click', () => pickFolderInto('modal_outputDir'));
    // Click outside the modal panel closes it
    document.getElementById('jobModal').addEventListener('click', (e) => {
        if (e.target.id === 'jobModal') closeModal();
    });
}

function openModal(index) {
    if (isRunning) {
        showToast(t('toast.error'), 'error');
        return;
    }
    editingJobIndex = index;
    const title = document.getElementById('modalTitle');
    if (index >= 0) {
        title.textContent = t('jobs.modal.edit_title');
        fillModalFromJob(jobs[index]);
    } else {
        title.textContent = t('jobs.modal.add_title');
        fillModalFromJob(defaultJob());
    }
    document.getElementById('jobModal').style.display = 'flex';
    // Focus the URL field so the user can start typing immediately.
    setTimeout(() => document.getElementById('modal_url').focus(), 50);
}

function closeModal() {
    document.getElementById('jobModal').style.display = 'none';
    editingJobIndex = -1;
}

function defaultJob() {
    return {
        url: '',
        output_dir: '',
        workers: 5,
        max_urls: 10000,
        max_depth: 0,
        timeout: 60,
        same_domain: true,
        allow_subdomains: true,
        skip_assets: false,
        insecure_tls: false,
        manifest: true,
        proxy: '',
        user_agent: '',
        cookies: '',
        allowed_hosts: '',
        asset_ext: '',
        status: 'pending'
    };
}

function fillModalFromJob(job) {
    document.getElementById('modal_url').value = job.url || '';
    document.getElementById('modal_outputDir').value = job.output_dir || '';
    document.getElementById('modal_workers').value = job.workers || 5;
    document.getElementById('modal_maxURLs').value = job.max_urls ?? 10000;
    document.getElementById('modal_maxDepth').value = job.max_depth ?? 0;
    document.getElementById('modal_timeout').value = job.timeout || 60;
    document.getElementById('modal_sameDomain').checked = job.same_domain !== false;
    document.getElementById('modal_allowSubs').checked = job.allow_subdomains !== false;
    document.getElementById('modal_skipAssets').checked = !!job.skip_assets;
    document.getElementById('modal_insecureTLS').checked = !!job.insecure_tls;
    document.getElementById('modal_manifest').checked = job.manifest !== false;
    document.getElementById('modal_proxy').value = job.proxy || '';
    document.getElementById('modal_userAgent').value = job.user_agent || '';
    document.getElementById('modal_cookies').value = job.cookies || '';
    document.getElementById('modal_allowedHosts').value = job.allowed_hosts || '';
    document.getElementById('modal_assetExt').value = job.asset_ext || '';
}

function readJobFromModal() {
    return {
        url: document.getElementById('modal_url').value.trim(),
        output_dir: document.getElementById('modal_outputDir').value.trim(),
        workers: parseInt(document.getElementById('modal_workers').value) || 5,
        max_urls: parseInt(document.getElementById('modal_maxURLs').value) || 10000,
        max_depth: parseInt(document.getElementById('modal_maxDepth').value) || 0,
        timeout: parseInt(document.getElementById('modal_timeout').value) || 60,
        same_domain: document.getElementById('modal_sameDomain').checked,
        allow_subdomains: document.getElementById('modal_allowSubs').checked,
        skip_assets: document.getElementById('modal_skipAssets').checked,
        insecure_tls: document.getElementById('modal_insecureTLS').checked,
        manifest: document.getElementById('modal_manifest').checked,
        proxy: document.getElementById('modal_proxy').value.trim(),
        user_agent: document.getElementById('modal_userAgent').value.trim(),
        cookies: document.getElementById('modal_cookies').value.trim(),
        allowed_hosts: document.getElementById('modal_allowedHosts').value.trim(),
        asset_ext: document.getElementById('modal_assetExt').value.trim(),
        status: 'pending'
    };
}

function saveJobFromModal() {
    const job = readJobFromModal();
    if (!job.url) {
        showToast(t('jobs.need_url'), 'error');
        return;
    }
    if (!job.output_dir) {
        showToast(t('jobs.need_output'), 'error');
        return;
    }
    if (editingJobIndex >= 0) {
        jobs[editingJobIndex] = job;
        showToast(t('toast.job_updated'), 'success');
    } else {
        jobs.push(job);
        showToast(t('toast.job_added'), 'success');
    }
    saveJobs();
    renderJobs();
    closeModal();
}

// ===== Start All Jobs =====
async function startAllJobs() {
    if (isRunning) return;
    if (jobs.length === 0) {
        showToast(t('jobs.need_jobs'), 'error');
        return;
    }
    // Validate every job has the required fields before sending.
    for (let i = 0; i < jobs.length; i++) {
        if (!jobs[i].url) {
            showToast(`Job ${i + 1}: ${t('jobs.need_url')}`, 'error');
            return;
        }
        if (!jobs[i].output_dir) {
            showToast(`Job ${i + 1}: ${t('jobs.need_output')}`, 'error');
            return;
        }
    }
    try {
        const res = await fetch('/api/jobs/start', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ jobs: jobs })
        });
        if (!res.ok) {
            const err = await res.json();
            showToast(err.error || t('error.crawl_failed'), 'error');
            return;
        }
        isRunning = true;
        isPaused = false;
        currentJobIndex = -1;
        totalJobs = jobs.length;
        startTime = Date.now();
        lastSampleBytes = 0;
        lastSampleTime = Date.now();
        peakSpeed = 0;
        speedSamples = [];
        for (const j of jobs) j.status = 'pending';
        renderJobs();
        showToast(t('toast.started'), 'success');
        setStatus('running');
        startElapsedTimer();
        startSpeedTimer();
        startEventSource();
        updateRunButtons();
    } catch (e) {
        showToast(e.message, 'error');
    }
}

// ===== Controls =====
function setupControls() {
    document.getElementById('pauseBtn').addEventListener('click', togglePause);
    document.getElementById('stopBtn').addEventListener('click', stopCrawl);
    document.getElementById('openOutputBtn').addEventListener('click', openOutput);
    document.getElementById('serveBtn').addEventListener('click', serveOutput);
}

// pickFolderInto opens the native OS folder-picker dialog and writes the
// selected path into the input identified by inputId. If the user cancels
// or no picker is available, the input is left unchanged.
//
// The backend /api/browse endpoint shells out to the platform's native
// dialog (PowerShell on Windows, osascript on macOS, zenity/kdialog on
// Linux). The HTTP request blocks until the user picks a folder or cancels,
// so we show a "..." state on the button while waiting.
async function pickFolderInto(inputId) {
    const input = document.getElementById(inputId);
    if (!input) return;
    // Find the sibling browse button (if any) and show a loading state.
    const btn = input.parentElement.querySelector('button');
    const originalText = btn ? btn.textContent : '';
    if (btn) {
        btn.disabled = true;
        btn.textContent = '...';
    }
    try {
        const body = input.value
            ? JSON.stringify({ current: input.value })
            : '{}';
        const res = await fetch('/api/browse', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: body
        });
        if (!res.ok) {
            showToast(t('toast.error'), 'error');
            return;
        }
        const data = await res.json();
        if (data.error) {
            // No picker available on this OS — fall back to a manual prompt
            // so the user isn't stuck.
            const fallback = prompt(
                currentLang === 'fa'
                    ? 'انتخابگر پوشه در دسترس نیست. مسیر را دستی وارد کنید:'
                    : 'No folder picker available. Enter path manually:',
                input.value
            );
            if (fallback) input.value = fallback;
            return;
        }
        if (data.path) {
            input.value = data.path;
        }
        // If path is empty, the user cancelled — do nothing.
    } catch (e) {
        showToast(e.message, 'error');
    } finally {
        if (btn) {
            btn.disabled = false;
            btn.textContent = originalText;
        }
    }
}

async function stopCrawl() {
    try {
        await fetch('/api/stop', { method: 'POST' });
        showToast(t('toast.stopped'), 'info');
        onQueueEnded();
    } catch (e) {
        showToast(e.message, 'error');
    }
}

async function togglePause() {
    const endpoint = isPaused ? '/api/resume' : '/api/pause';
    try {
        const res = await fetch(endpoint, { method: 'POST' });
        if (!res.ok) {
            const err = await res.json();
            showToast(err.error || t('toast.error'), 'error');
            return;
        }
        isPaused = !isPaused;
        updatePauseButtonLabel();
        if (isPaused) {
            setStatus('paused');
            showToast(t('toast.paused'), 'info');
        } else {
            setStatus('running');
            showToast(t('toast.resumed'), 'info');
        }
        renderJobs();
    } catch (e) {
        showToast(e.message, 'error');
    }
}

function updatePauseButtonLabel() {
    const btn = document.getElementById('pauseBtn');
    if (!btn) return;
    const span = btn.querySelector('span');
    if (!span) return;
    span.textContent = isPaused ? t('controls.resume') : t('controls.pause');
}

function updateRunButtons() {
    document.getElementById('pauseBtn').disabled = !isRunning;
    document.getElementById('stopBtn').disabled = !isRunning;
}

async function openOutput() {
    const dir = document.getElementById('outputDir').value.trim();
    if (!dir) return;
    try {
        await fetch('/api/open', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ path: dir })
        });
    } catch (e) {}
}

async function serveOutput() {
    const dir = document.getElementById('outputDir').value.trim();
    if (!dir) return;
    try {
        const res = await fetch('/api/serve', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ path: dir })
        });
        const data = await res.json();
        if (data.url) {
            window.open(data.url, '_blank');
        }
    } catch (e) {}
}

// ===== Status recovery on page load =====
//
// If the user refreshes the page while a queue is running, we want to
// re-attach to the running queue rather than show an idle UI. /api/status
// tells us the current queue state (which job is running, how many total,
// the counter snapshot, etc.) so we can restore everything.
async function pollStatus() {
    try {
        const res = await fetch('/api/status');
        if (!res.ok) return;
        const data = await res.json();
        if (data.queue_state === 'running' || data.queue_state === 'paused') {
            isRunning = true;
            isPaused = (data.queue_state === 'paused');
            currentJobIndex = data.queue_index ?? -1;
            totalJobs = data.queue_total || 0;
            startTime = Date.now(); // best-effort; backend doesn't expose start time
            // Populate stat cards from the snapshot so the user sees
            // continuity rather than zeros.
            document.getElementById('statPages').textContent = data.pages || 0;
            document.getElementById('statAssets').textContent = data.assets || 0;
            document.getElementById('statBytes').textContent = humanBytes(data.bytes || 0);
            document.getElementById('statFailed').textContent = data.failed || 0;
            document.getElementById('statSkipped').textContent = data.skipped || 0;
            lastSampleBytes = data.bytes || 0;
            lastSampleTime = Date.now();
            // If we have a running queue but no saved jobs in localStorage,
            // we still want to show SOMETHING in the jobs tab — synthesize a
            // minimal placeholder list from the queue info.
            if (jobs.length === 0 && data.queue_url) {
                jobs = [{ url: data.queue_url, output_dir: '', status: 'running' }];
            }
            setStatus(isPaused ? 'paused' : 'running');
            startElapsedTimer();
            startSpeedTimer();
            startEventSource();
            renderJobs();
            updateRunButtons();
            updatePauseButtonLabel();
            updateCurrentJobCard();
        }
    } catch (e) {}
}

// ===== SSE =====
function startEventSource() {
    if (eventSource) eventSource.close();
    eventSource = new EventSource('/api/events');
    eventSource.onmessage = (e) => {
        try {
            const ev = JSON.parse(e.data);
            handleEvent(ev);
        } catch (err) {}
    };
    eventSource.onerror = () => {
        // Browser auto-reconnects; nothing to do here.
    };
}

function handleEvent(ev) {
    // Update job_index tracking so the right job card lights up.
    if (ev.job_total && ev.job_total > 0) {
        totalJobs = ev.job_total;
    }
    if (ev.job_index !== undefined && ev.job_index !== null && ev.job_index >= 0) {
        if (ev.job_index !== currentJobIndex) {
            // The active job just changed — mark the previous one as done
            // (the backend only advances after a job finishes) and bump.
            if (currentJobIndex >= 0 && currentJobIndex < jobs.length) {
                jobs[currentJobIndex].status = 'done';
            }
            currentJobIndex = ev.job_index;
            if (currentJobIndex < jobs.length) {
                jobs[currentJobIndex].status = 'running';
            }
            // Reset per-job counters display so they don't bleed across jobs.
            resetStatCards();
            startTime = Date.now();
            lastSampleBytes = 0;
            lastSampleTime = Date.now();
            renderJobs();
            updateCurrentJobCard();
        }
    }

    // Update counters
    if (ev.visited !== undefined && ev.visited > 0) {
        document.getElementById('statPages').textContent = ev.pages || 0;
        document.getElementById('statAssets').textContent = ev.assets || 0;
        document.getElementById('statBytes').textContent = humanBytes(ev.total_bytes || 0);
        document.getElementById('statFailed').textContent = ev.failed || 0;
        document.getElementById('statSkipped').textContent = ev.skipped || 0;

        // Progress bar is per-job: visited / max_urls of the CURRENT job.
        if (currentJobIndex >= 0 && currentJobIndex < jobs.length) {
            const maxURLs = jobs[currentJobIndex].max_urls || 10000;
            if (maxURLs > 0) {
                const pct = Math.min(100, (ev.visited / maxURLs) * 100);
                document.getElementById('progressFill').style.width = pct + '%';
                document.getElementById('progressText').textContent = Math.round(pct) + '%';
            }
        }
    }

    // Pause / resume events can come from the backend (e.g. another tab
    // clicked pause). Keep our local flag in sync.
    if (ev.type === 'pause') {
        isPaused = true;
        updatePauseButtonLabel();
        setStatus('paused');
        renderJobs();
        addLog(formatLogLine(ev), 'warn');
        return;
    }
    if (ev.type === 'resume') {
        isPaused = false;
        updatePauseButtonLabel();
        setStatus('running');
        renderJobs();
        addLog(formatLogLine(ev), 'success');
        return;
    }

    if (ev.url) {
        document.getElementById('currentURL').textContent = ev.url;
        document.getElementById('footerURL').textContent = ev.url;
    }
    if (ev.path) {
        document.getElementById('lastSaved').textContent = ev.path;
    }
    if (ev.msg || ev.url) {
        addLog(formatLogLine(ev), ev.type);
    }

    // End event — the whole queue finished (not just one job).
    if (ev.type === 'end') {
        // Mark the current job as done.
        if (currentJobIndex >= 0 && currentJobIndex < jobs.length) {
            jobs[currentJobIndex].status = 'done';
        }
        setStatus('done');
        showToast(t('toast.completed'), 'success');
        onQueueEnded();
    }
}

function resetStatCards() {
    document.getElementById('statPages').textContent = '0';
    document.getElementById('statAssets').textContent = '0';
    document.getElementById('statBytes').textContent = '0 B';
    document.getElementById('statFailed').textContent = '0';
    document.getElementById('statSkipped').textContent = '0';
    document.getElementById('progressFill').style.width = '0%';
    document.getElementById('progressText').textContent = '0%';
}

function updateCurrentJobCard() {
    const card = document.getElementById('currentJobCard');
    if (!card) return;
    if (!isRunning || currentJobIndex < 0) {
        card.style.display = 'none';
        return;
    }
    card.style.display = 'block';
    document.getElementById('currentJobIndex').textContent =
        `${currentJobIndex + 1} / ${totalJobs}`;
    const url = (currentJobIndex < jobs.length) ? jobs[currentJobIndex].url : '—';
    document.getElementById('currentJobURL').textContent = url;
}

function onQueueEnded() {
    isRunning = false;
    isPaused = false;
    currentJobIndex = -1;
    stopElapsedTimer();
    stopSpeedTimer();
    if (eventSource) {
        eventSource.close();
        eventSource = null;
    }
    updateRunButtons();
    updatePauseButtonLabel();
    updateCurrentJobCard();
    renderJobs();
}

function formatLogLine(ev) {
    const ts = new Date(ev.time).toLocaleTimeString();
    const prefix = (ev.job_total && ev.job_total > 1 && ev.job_index >= 0)
        ? `[J${ev.job_index + 1}/${ev.job_total}] `
        : '';
    switch (ev.type) {
        case 'start': return `[${ts}] ${prefix}START  ${ev.msg}`;
        case 'fetch_start': return `[${ts}] ${prefix}FETCH  ${ev.url}`;
        case 'fetch_ok': return `[${ts}] ${prefix}OK     ${ev.url} (${ev.bytes} bytes)`;
        case 'fetch_error': return `[${ts}] ${prefix}ERR    ${ev.url} — ${ev.msg}`;
        case 'save': return `[${ts}] ${prefix}SAVE   ${ev.msg} [${ev.path}] (${ev.bytes} bytes)`;
        case 'pause': return `[${ts}] ${prefix}PAUSE  ${ev.msg}`;
        case 'resume': return `[${ts}] ${prefix}RESUME ${ev.msg}`;
        case 'end': return `[${ts}] ${prefix}END    ${ev.msg}`;
        case 'log': return `[${ts}] ${prefix}${ev.msg}`;
    }
    return `[${ts}] ${prefix}${ev.msg || ev.url || ''}`;
}

// ===== Logs =====
function setupLogs() {
    document.getElementById('clearLogsBtn').addEventListener('click', () => {
        allLogs = [];
        renderLogs();
    });
    document.getElementById('copyLogsBtn').addEventListener('click', () => {
        const text = allLogs.map(l => l.line).join('\n');
        navigator.clipboard.writeText(text).then(() => {
            showToast(t('toast.copied'), 'success');
        });
    });
    document.getElementById('logFilter').addEventListener('input', renderLogs);
}

function addLog(line, type) {
    allLogs.push({ line, type });
    if (allLogs.length > 5000) allLogs = allLogs.slice(-5000);
    renderLogs();
}

function renderLogs() {
    const filter = document.getElementById('logFilter').value.toLowerCase().trim();
    const viewer = document.getElementById('logsViewer');
    const filtered = filter
        ? allLogs.filter(l => l.line.toLowerCase().includes(filter))
        : allLogs;
    const html = filtered.map(l => {
        let cls = '';
        if (l.type === 'fetch_error') cls = 'error';
        else if (l.type === 'save') cls = 'success';
        else if (l.type === 'pause' || l.type === 'resume') cls = 'warn';
        return `<div class="log-line ${cls}">${escapeHtml(l.line)}</div>`;
    }).join('');
    viewer.innerHTML = html;
    viewer.scrollTop = viewer.scrollHeight;
}

function escapeHtml(s) {
    const div = document.createElement('div');
    div.textContent = s;
    return div.innerHTML;
}

// ===== Status display =====
function setStatus(status) {
    const dot = document.getElementById('statusDot');
    const text = document.getElementById('statusText');
    const footer = document.getElementById('footerStatus');
    dot.className = 'status-dot';
    if (status === 'running') {
        dot.classList.add('running');
        text.textContent = t('controls.status_running');
        footer.textContent = t('controls.status_running');
    } else if (status === 'paused') {
        dot.classList.add('paused');
        text.textContent = t('controls.status_paused');
        footer.textContent = t('controls.status_paused');
    } else if (status === 'done') {
        dot.classList.add('done');
        text.textContent = t('controls.status_done');
        footer.textContent = t('controls.status_done');
    } else if (status === 'failed') {
        dot.classList.add('error');
        text.textContent = t('controls.status_failed');
        footer.textContent = t('controls.status_failed');
    } else {
        text.textContent = t('controls.status_idle');
        footer.textContent = t('controls.status_idle');
    }
}

// ===== Elapsed timer =====
function startElapsedTimer() {
    stopElapsedTimer();
    elapsedTimer = setInterval(() => {
        if (!startTime) return;
        const elapsed = Math.floor((Date.now() - startTime) / 1000);
        const m = Math.floor(elapsed / 60);
        const s = elapsed % 60;
        document.getElementById('statElapsed').textContent =
            `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`;
    }, 1000);
}

function stopElapsedTimer() {
    if (elapsedTimer) {
        clearInterval(elapsedTimer);
        elapsedTimer = null;
    }
}

// ===== Speed chart =====
function setupSpeedChart() {
    const canvas = document.getElementById('speedChart');
    if (!canvas) return;
    canvasCtx = canvas.getContext('2d');
    const dpr = window.devicePixelRatio || 1;
    const rect = canvas.getBoundingClientRect();
    canvas.width = rect.width * dpr;
    canvas.height = rect.height * dpr;
    canvasCtx.scale(dpr, dpr);
    drawSpeedChart();
}

function startSpeedTimer() {
    stopSpeedTimer();
    speedTimer = setInterval(sampleSpeed, SPEED_SAMPLE_MS);
}

function stopSpeedTimer() {
    if (speedTimer) {
        clearInterval(speedTimer);
        speedTimer = null;
    }
}

function sampleSpeed() {
    const bytesEl = document.getElementById('statBytes');
    if (!bytesEl) return;
    const totalBytes = parseHumanBytes(bytesEl.textContent);
    const now = Date.now();
    const dt = (now - lastSampleTime) / 1000;
    if (dt <= 0) return;
    const delta = Math.max(0, totalBytes - lastSampleBytes);
    const bps = Math.round(delta / dt);
    if (bps > peakSpeed) peakSpeed = bps;

    speedSamples.push({ t: now, bytes: totalBytes, bps });
    if (speedSamples.length > SPEED_MAX_SAMPLES) {
        speedSamples.shift();
    }

    lastSampleBytes = totalBytes;
    lastSampleTime = now;

    document.getElementById('statSpeed').textContent = humanBytes(bps) + '/s';
    const totalElapsed = startTime ? (now - startTime) / 1000 : 0;
    if (totalElapsed > 0) {
        const avg = Math.round(totalBytes / totalElapsed);
        document.getElementById('statAvgSpeed').textContent = humanBytes(avg) + '/s';
    }
    document.getElementById('statPeakSpeed').textContent = humanBytes(peakSpeed) + '/s';
    drawSpeedChart();
}

function parseHumanBytes(s) {
    if (!s) return 0;
    const m = s.trim().match(/^([\d.]+)\s*(B|KB|MB|GB|TB)?/i);
    if (!m) return 0;
    const n = parseFloat(m[1]);
    const unit = (m[2] || 'B').toUpperCase();
    switch (unit) {
        case 'B': return Math.round(n);
        case 'KB': return Math.round(n * 1024);
        case 'MB': return Math.round(n * 1024 * 1024);
        case 'GB': return Math.round(n * 1024 * 1024 * 1024);
        case 'TB': return Math.round(n * 1024 * 1024 * 1024 * 1024);
    }
    return Math.round(n);
}

function drawSpeedChart() {
    if (!canvasCtx) return;
    const canvas = document.getElementById('speedChart');
    const w = canvas.clientWidth;
    const h = canvas.clientHeight;
    canvasCtx.clearRect(0, 0, w, h);

    const gridColor = getComputedStyle(document.documentElement)
        .getPropertyValue('--chart-grid').trim();
    canvasCtx.strokeStyle = gridColor;
    canvasCtx.lineWidth = 1;
    for (let i = 1; i < 4; i++) {
        const y = (h / 4) * i;
        canvasCtx.beginPath();
        canvasCtx.moveTo(0, y);
        canvasCtx.lineTo(w, y);
        canvasCtx.stroke();
    }

    if (speedSamples.length < 2) return;

    const maxBps = Math.max(1024, ...speedSamples.map(s => s.bps));
    const lineColor = getComputedStyle(document.documentElement)
        .getPropertyValue('--chart-line').trim();
    const fillColor = getComputedStyle(document.documentElement)
        .getPropertyValue('--chart-fill').trim();

    const stepX = w / (SPEED_MAX_SAMPLES - 1);
    const startX = w - (speedSamples.length - 1) * stepX;

    canvasCtx.beginPath();
    canvasCtx.moveTo(startX, h);
    speedSamples.forEach((s, i) => {
        const x = startX + i * stepX;
        const y = h - (s.bps / maxBps) * (h - 8) - 4;
        canvasCtx.lineTo(x, y);
    });
    canvasCtx.lineTo(startX + (speedSamples.length - 1) * stepX, h);
    canvasCtx.closePath();
    canvasCtx.fillStyle = fillColor;
    canvasCtx.fill();

    canvasCtx.beginPath();
    speedSamples.forEach((s, i) => {
        const x = startX + i * stepX;
        const y = h - (s.bps / maxBps) * (h - 8) - 4;
        if (i === 0) canvasCtx.moveTo(x, y);
        else canvasCtx.lineTo(x, y);
    });
    canvasCtx.strokeStyle = lineColor;
    canvasCtx.lineWidth = 2;
    canvasCtx.stroke();
}

// ===== Helpers =====
function humanBytes(b) {
    if (b < 1024) return b + ' B';
    if (b < 1024 * 1024) return (b / 1024).toFixed(1) + ' KB';
    if (b < 1024 * 1024 * 1024) return (b / (1024 * 1024)).toFixed(1) + ' MB';
    return (b / (1024 * 1024 * 1024)).toFixed(2) + ' GB';
}

function showToast(msg, type = 'info') {
    const container = document.getElementById('toastContainer');
    const toast = document.createElement('div');
    toast.className = 'toast ' + type;
    toast.textContent = msg;
    container.appendChild(toast);
    setTimeout(() => {
        toast.style.opacity = '0';
        setTimeout(() => toast.remove(), 300);
    }, 3000);
}
