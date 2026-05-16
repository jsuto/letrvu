#!/usr/bin/env python3
"""Populate the GreenMail demo inbox with sample emails.

Runs in a loop: after seeding, polls every 60 s and resets immediately
whenever the inbox diverges from the known-good seed state (new message
added by a visitor, or a seed message deleted).
"""

import imaplib
import ssl
import json
import smtplib
import socket
import time
import urllib.request
from datetime import datetime, timezone, timedelta
from email.mime.multipart import MIMEMultipart
from email.mime.text import MIMEText
from email.mime.base import MIMEBase
from email import encoders
from email.utils import formatdate, make_msgid

SMTP_HOST = "greenmail"
SMTP_PORT = 3025
IMAP_HOST = "greenmail"
IMAP_PORT = 3993
GREENMAIL_API = "http://greenmail:8080/api"
DEMO_USER = "demo@letrvu.demo"
DEMO_PASS = "demo@letrvu.demo"
TO = DEMO_USER


# ── helpers ───────────────────────────────────────────────────────────────────

def ago(days=0, hours=0):
    dt = datetime.now(timezone.utc) - timedelta(days=days, hours=hours)
    return formatdate(dt.timestamp(), localtime=False)


def send(msg):
    with smtplib.SMTP(SMTP_HOST, SMTP_PORT) as s:
        s.sendmail(msg["From"], [TO], msg.as_bytes())
    print(f"  sent: {msg['Subject']!r}", flush=True)


def plain_msg(frm, subj, body, date, msg_id=None, in_reply_to=None, references=None):
    m = MIMEText(body, "plain", "utf-8")
    m["From"] = frm
    m["To"] = TO
    m["Subject"] = subj
    m["Date"] = date
    m["Message-ID"] = msg_id or make_msgid(domain="seed.letrvu.demo")
    if in_reply_to:
        m["In-Reply-To"] = in_reply_to
    if references:
        m["References"] = references
    return m


def html_msg(frm, subj, html, plain, date, msg_id=None, in_reply_to=None, references=None):
    m = MIMEMultipart("alternative")
    m["From"] = frm
    m["To"] = TO
    m["Subject"] = subj
    m["Date"] = date
    m["Message-ID"] = msg_id or make_msgid(domain="seed.letrvu.demo")
    if in_reply_to:
        m["In-Reply-To"] = in_reply_to
    if references:
        m["References"] = references
    m.attach(MIMEText(plain, "plain", "utf-8"))
    m.attach(MIMEText(html, "html", "utf-8"))
    return m


def with_attachment(frm, subj, body, date, filename, data, mimetype="application/octet-stream"):
    m = MIMEMultipart()
    m["From"] = frm
    m["To"] = TO
    m["Subject"] = subj
    m["Date"] = date
    m["Message-ID"] = make_msgid(domain="seed.letrvu.demo")
    m.attach(MIMEText(body, "plain", "utf-8"))
    main, sub = mimetype.split("/", 1)
    att = MIMEBase(main, sub)
    att.set_payload(data)
    encoders.encode_base64(att)
    att.add_header("Content-Disposition", "attachment", filename=filename)
    m.attach(att)
    return m


# ── GreenMail helpers ─────────────────────────────────────────────────────────

def wait_for_imap():
    print("Waiting for GreenMail...", flush=True)
    for _ in range(60):
        try:
            with socket.create_connection((IMAP_HOST, IMAP_PORT), timeout=2):
                pass
            # Also wait for the HTTP API used by wipe_inbox().
            urllib.request.urlopen(f"{GREENMAIL_API}/service/readiness", timeout=2)
            print("GreenMail ready.", flush=True)
            return
        except Exception:
            time.sleep(1)
    raise SystemExit("GreenMail did not become available.")


def ensure_folders():
    """Create standard IMAP folders GreenMail doesn't auto-provision."""
    try:
        m = _imap_connect()
        m.login(DEMO_USER, DEMO_PASS)
        for folder in ("Drafts", "Sent", "Trash", "Junk"):
            m.create(folder)
        m.logout()
    except Exception as e:
        print(f"Note: folder setup: {e}", flush=True)


def wipe_inbox():
    """Purge all messages via the GreenMail API."""
    req = urllib.request.Request(f"{GREENMAIL_API}/service/reset", method="POST")
    try:
        urllib.request.urlopen(req, timeout=5)
        print("Inbox wiped.", flush=True)
    except Exception as e:
        print(f"Warning: could not wipe inbox: {e}", flush=True)


def _imap_connect():
    ctx = ssl.create_default_context()
    ctx.check_hostname = False
    ctx.verify_mode = ssl.CERT_NONE
    return imaplib.IMAP4_SSL(IMAP_HOST, IMAP_PORT, ssl_context=ctx)


def get_inbox_uids():
    """Return the set of IMAP UIDs in the INBOX, or None on error."""
    try:
        m = _imap_connect()
        m.login(DEMO_USER, DEMO_PASS)
        m.select("INBOX", readonly=True)
        _, data = m.uid("search", None, "ALL")
        m.logout()
        uids = set(data[0].split()) if data and data[0] else set()
        return uids
    except Exception as e:
        print(f"Warning: IMAP check failed: {e}", flush=True)
        return None


def get_inbox_count():
    """Return the number of messages in INBOX, or None on error."""
    uids = get_inbox_uids()
    return len(uids) if uids is not None else None


def watch(seed_count):
    """Poll every 60 s.

    - Extra messages (visitor sent mail): delete them via IMAP.
    - Fewer messages than seeded: return so the caller triggers a full re-seed.
    """
    print("Watching inbox for changes...", flush=True)
    while True:
        time.sleep(60)
        try:
            m = _imap_connect()
            m.login(DEMO_USER, DEMO_PASS)
            m.select("INBOX")
            _, data = m.uid("search", None, "ALL")
            uids = data[0].split() if data and data[0] else []
            count = len(uids)

            if count < seed_count:
                m.logout()
                print("Seed message deleted — re-seeding.", flush=True)
                return

            if count > seed_count:
                # Delete messages beyond the seed set (newest first).
                extra = uids[seed_count:]
                for uid in extra:
                    m.uid("store", uid, "+FLAGS", "\\Deleted")
                    print(f"  removed foreign message uid={uid.decode()}", flush=True)
                m.expunge()

            m.logout()
        except Exception as e:
            print(f"Warning: watch error: {e}", flush=True)


# ── seed data ─────────────────────────────────────────────────────────────────

def seed_and_watch():
    wipe_inbox()
    print("Seeding demo inbox...")

    # 1. Welcome email ─────────────────────────────────────────────────────────
    send(html_msg(
        frm="Letrvu <team@letrvu.io>",
        subj="Welcome to your Letrvu demo!",
        date=ago(days=14),
        html="""\
<div style="font-family:sans-serif;max-width:560px;margin:0 auto;color:#111">
  <h2 style="color:#2563eb;margin-bottom:4px">Welcome to Letrvu 👋</h2>
  <p>This is a sandboxed demo inbox — feel free to explore. Everything you see here
  was generated just for this demo; emails you send stay within this environment.</p>
  <h3 style="margin-bottom:6px">What you can try</h3>
  <ul style="line-height:1.8">
    <li>Read and reply to the emails already in your inbox</li>
    <li>Compose a new message (it will land back here)</li>
    <li>Add contacts and import a vCard</li>
    <li>Open <strong>Settings → Security</strong> to generate a PGP key and sign outbound mail</li>
  </ul>
  <p style="color:#555;font-size:13px">
    Letrvu is open-source and self-hosted — it connects to any standard IMAP/SMTP
    server. No mail server is bundled; this demo uses
    <a href="https://greenmail-project.github.io/greenmail/">GreenMail</a> as a local sandbox.
  </p>
  <p>— The Letrvu Team</p>
</div>""",
        plain="""\
Welcome to Letrvu!

This is a sandboxed demo inbox. Emails you send stay within this environment.

What you can try:
- Read and reply to the emails already in your inbox
- Compose a new message (it will land back here)
- Add contacts and import a vCard
- Open Settings → Security to generate a PGP key

Letrvu is open-source and self-hosted. This demo uses GreenMail as a local sandbox.

— The Letrvu Team""",
    ))

    # 2. Invoice with attachment ────────────────────────────────────────────────
    invoice_txt = ("""\
INVOICE #2041
=============
Date:    %s
Bill to: demo@letrvu.demo

  1x Infra-Co Developer Plan (monthly)   $49.00
  ----------------------------------------
  Total due:                              $49.00

Payment method: Visa ending 4242
Thank you for your business.
""" % ago(days=10)).encode()

    send(with_attachment(
        frm="Billing <billing@infra-co.example>",
        subj="Invoice #2041 — Infra-Co Developer Plan",
        date=ago(days=10),
        body="""\
Hi,

Please find your invoice for this month attached.

Amount due: $49.00
Due date:   net 14 days

You can view your billing history at https://infra-co.example/billing

Thanks,
Infra-Co Billing""",
        filename="invoice-2041.txt",
        data=invoice_txt,
        mimetype="text/plain",
    ))

    # 3. Thread: deployment question ───────────────────────────────────────────
    thread_id = make_msgid(domain="seed.letrvu.demo")

    send(plain_msg(
        frm="Alex Kovacs <alex@example.com>",
        subj="Deployment question",
        date=ago(days=8),
        msg_id=thread_id,
        body="""\
Hey,

Quick question — are you deploying Letrvu behind a reverse proxy?
I'm trying to set it up with Caddy and the session cookies aren't
surviving the proxy hop. The SECURE_COOKIES env var is set to true
but I'm getting logged out on every request.

Any idea what I'm missing?

Cheers,
Alex""",
    ))

    send(plain_msg(
        frm="Alex Kovacs <alex@example.com>",
        subj="Re: Deployment question",
        date=ago(days=7),
        in_reply_to=thread_id,
        references=thread_id,
        body="""\
Figured it out — I forgot to set TRUSTED_PROXY to Caddy's container IP.
Once I added that, X-Forwarded-Proto is trusted and the cookies work fine.

Leaving this here in case anyone else runs into it.

Alex""",
    ))

    # 4. GitHub-style PR notification ──────────────────────────────────────────
    send(html_msg(
        frm="GitHub <notifications@github.com>",
        subj="[letrvu/letrvu] PGP/MIME support merged into main (#42)",
        date=ago(days=5),
        html="""\
<div style="font-family:sans-serif;font-size:14px;color:#24292f;max-width:600px">
  <table width="100%" cellpadding="0" cellspacing="0">
    <tr>
      <td style="padding:16px 0;border-bottom:1px solid #d0d7de">
        <strong>letrvu / letrvu</strong>
      </td>
    </tr>
  </table>
  <div style="padding:16px 0">
    <p>
      <span style="background:#8250df;color:#fff;border-radius:12px;padding:3px 10px;font-size:12px">merged</span>
      &nbsp;
      <strong>feat: PGP/MIME signing and WKD key discovery</strong>
      &nbsp;
      <a href="#" style="color:#0969da">#42</a>
    </p>
    <p style="color:#57606a">
      <strong>sutoj</strong> merged 14 commits into <code>main</code> from
      <code>feature/pgp-mime</code>
    </p>
    <ul style="color:#57606a;line-height:1.8">
      <li>RFC 3156 multipart/signed detached signatures</li>
      <li>WKD auto-lookup for recipient public keys</li>
      <li>Per-contact PGP key storage</li>
      <li>Signature verification banner in message view</li>
    </ul>
  </div>
  <p style="color:#57606a;font-size:12px">
    You are receiving this because you are subscribed to this repository.
  </p>
</div>""",
        plain="""\
[letrvu/letrvu] feat: PGP/MIME signing and WKD key discovery (#42) merged into main

sutoj merged 14 commits into main from feature/pgp-mime

- RFC 3156 multipart/signed detached signatures
- WKD auto-lookup for recipient public keys
- Per-contact PGP key storage
- Signature verification banner in message view

You are receiving this because you are subscribed to this repository.""",
    ))

    # 5. Newsletter ────────────────────────────────────────────────────────────
    send(html_msg(
        frm="TLDR Newsletter <newsletter@tldr.example>",
        subj="TLDR — Top developer stories this week",
        date=ago(days=3),
        html="""\
<div style="font-family:sans-serif;max-width:560px;margin:0 auto;color:#111">
  <h2 style="border-bottom:2px solid #2563eb;padding-bottom:8px">
    TLDR &mdash; Developer digest
  </h2>

  <h3>🔧 Tools &amp; Open Source</h3>
  <p>
    <strong><a href="#" style="color:#2563eb">Letrvu — self-hosted webmail with PGP support</a></strong><br>
    <span style="color:#555">
      A new open-source webmail client built with Go and Vue 3. Connects to
      any IMAP/SMTP server, ships as a single binary, and includes browser-side
      PGP signing and WKD key discovery.
    </span>
  </p>

  <h3>📖 Articles</h3>
  <p>
    <strong><a href="#" style="color:#2563eb">Why self-hosting email clients is making a comeback</a></strong><br>
    <span style="color:#555">Privacy concerns and vendor lock-in are pushing developers back to
    self-hosted tooling for mission-critical workflows.</span>
  </p>
  <p>
    <strong><a href="#" style="color:#2563eb">OpenPGP in 2025: WKD, key servers, and what actually works</a></strong><br>
    <span style="color:#555">A practical look at the current state of PGP key discovery and
    which methods interoperate reliably with major providers.</span>
  </p>

  <p style="color:#888;font-size:12px;margin-top:32px">
    You subscribed to this newsletter.
    <a href="#" style="color:#888">Unsubscribe</a>
  </p>
</div>""",
        plain="""\
TLDR — Developer digest

🔧 Tools & Open Source
Letrvu — self-hosted webmail with PGP support
A new open-source webmail client built with Go and Vue 3. Connects to any
IMAP/SMTP server, ships as a single binary, and includes browser-side PGP
signing and WKD key discovery.

📖 Articles
Why self-hosting email clients is making a comeback
Privacy concerns and vendor lock-in are pushing developers back to self-hosted
tooling for mission-critical workflows.

OpenPGP in 2025: WKD, key servers, and what actually works
A practical look at the current state of PGP key discovery and which methods
interoperate reliably with major providers.""",
    ))

    # 6. Plain-text note from a friend ─────────────────────────────────────────
    send(plain_msg(
        frm="Maria Santos <maria@example.com>",
        subj="saw your project on HN",
        date=ago(days=1),
        body="""\
Hey!

Saw Letrvu pop up on the HN front page — congrats, that's a big deal.
I tried it out with my Fastmail account and the setup took maybe two minutes.
PGP key generation from the browser is a nice touch, I wasn't expecting that.

One thing: on mobile the compose button was a bit hard to hit. Might be
worth making it bigger. Otherwise looks really solid.

Let me know if you need beta testers.

Maria""",
    ))

    # 7. Security notice ───────────────────────────────────────────────────────
    send(plain_msg(
        frm="Fastmail Security <security@fastmail.com>",
        subj="Reminder: enable two-factor authentication",
        date=ago(hours=3),
        body="""\
Hi,

We noticed your account doesn't have two-factor authentication enabled.
Accounts with 2FA are significantly more resistant to unauthorised access.

To enable 2FA: Settings → Privacy & Security → Two-step verification

If you've already set this up, you can ignore this reminder.

— The Fastmail Security Team

This is an automated message. Please do not reply.""",
    ))

    # User now exists in GreenMail (created on first SMTP delivery above).
    ensure_folders()

    print("Done — demo inbox seeded with 7 emails.", flush=True)

    seed_count = get_inbox_count()
    if seed_count is not None:
        watch(seed_count)


# ── main loop ─────────────────────────────────────────────────────────────────

wait_for_imap()

while True:
    seed_and_watch()
