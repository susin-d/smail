import sys
import subprocess

def install_and_import():
    try:
        from fpdf import FPDF
    except ImportError:
        subprocess.check_call([sys.executable, "-m", "pip", "install", "fpdf2"])
        from fpdf import FPDF
    return FPDF

FPDF = install_and_import()

class PDF(FPDF):
    def header(self):
        self.set_font('helvetica', 'B', 15)
        self.cell(0, 10, 'MaaS Backend Resource Usage & Scalability Report', 0, 1, 'C')
        self.ln(5)

    def footer(self):
        self.set_y(-15)
        self.set_font('helvetica', 'I', 8)
        self.cell(0, 10, f'Page {self.page_no()}', 0, 0, 'C')

    def chapter_title(self, title):
        self.set_font('helvetica', 'B', 12)
        self.set_fill_color(200, 220, 255)
        self.cell(0, 8, title, 0, 1, 'L', fill=True)
        self.ln(4)

    def chapter_body(self, body):
        self.set_font('helvetica', '', 10)
        self.multi_cell(0, 6, body)
        self.ln(4)

pdf = PDF()
pdf.add_page()
pdf.set_auto_page_break(auto=True, margin=15)

body_1 = """The Mail-as-a-Service (MaaS) backend has been highly optimized to run within a strict 1GB RAM constraint. The architecture utilizes Go for the API layer due to its minimal footprint and high concurrency capabilities, alongside finely tuned Docker containers for Postfix, Dovecot, Rspamd, MariaDB, and Redis.

Below is the theoretical resource breakdown based on our configured Docker Compose limits and expected application behavior."""
pdf.chapter_title('1. Architecture & Baseline Overheads')
pdf.chapter_body(body_1)

body_2 = """- MariaDB (mem_limit: 80m): Pre-allocated buffer pools and connection limits.
- Redis (mem_limit: 50m): In-memory cache for Rspamd Bayes and Go API queues.
- Postfix (mem_limit: 100m): SMTP server with bounded process limits.
- Dovecot (mem_limit: 150m): IMAP server.
- OpenDKIM (mem_limit: 50m): Lightweight DKIM signer.
- Rspamd (mem_limit: 150m): Anti-spam engine.
- Go API (mem_limit: 150m): Compiled Go binary handling HTTP requests.
- Nginx (mem_limit: 30m): Reverse proxy.

TOTAL BASELINE (IDLE): ~650 - 700 MB.
The system is well within the 1GB hard limit while completely idle."""
pdf.chapter_title('2. Idle State (0 Active Users)')
pdf.chapter_body(body_2)

body_3 = """Assumptions: Occasional API requests, ~10 active IMAP IDLE connections, intermittent SMTP traffic.

- Go API: Memory footprint remains largely unchanged (~20-30MB base). Goroutines are extremely cheap (2KB each).
- Dovecot: IMAP connections cost roughly 1-2MB each. 10 active connections add ~15MB.
- Postfix/Rspamd: Brief, transient spikes during email delivery or spam checks (+20MB average).
- Database/Redis: Minimal changes.

TOTAL RAM: ~750 MB.
CPU USAGE: < 2% average. System handles this load with zero strain."""
pdf.chapter_title('3. Moderate Load (100 Users)')
pdf.chapter_body(body_3)

body_4 = """Assumptions: ~250 concurrent IMAP IDLE connections, frequent API polling, active SMTP queueing.

- Go API: HTTP traffic increases, garbage collector works more frequently. Go will use around 50-80MB RAM.
- Dovecot: 250 connections * 1.5MB = ~375MB. This is the largest memory consumer. We have configured vsz_limit to strictly cap Dovecot processes.
- Postfix/Rspamd: Increased SMTP transactions. Rspamd will constantly use CPU to evaluate spam. RAM hits its 150MB ceiling quickly.

TOTAL RAM: ~950 MB - 1.1 GB.
EXPECTATION: The system is heavily strained here solely due to the 1GB hard constraint. Dovecot IMAP processes may begin to get OOM killed or reject new connections if the memory limit is strictly enforced by Docker.
CPU USAGE: 15 - 30% average, spiking to 100% during bulk email sends."""
pdf.chapter_title('4. High Load (1,000 Users)')
pdf.chapter_body(body_4)

body_5 = """EDGE CASE A: Inbound Spam Attack (e.g., 10,000 emails/minute)
Postfix is configured to aggressively throttle connections (smtpd_client_connection_rate_limit = 10, smtpd_client_connection_count_limit = 5). Rspamd will max out its CPU quota, but memory is safely capped. Emails will queue on the disk or be rejected (4xx) at the gate, keeping the system online.

EDGE CASE B: Massive IMAP Connection Storm
If a network partition resolves and 1,000 clients attempt to reconnect to IMAP simultaneously, Dovecot's RAM footprint will attempt to instantly swell. Docker will OOM kill Dovecot processes to protect the host machine. Clients will experience connection failures until they back-off and retry, stabilizing the server.

EDGE CASE C: Large Attachment Processing
Sending a 10MB attachment requires the Go API to load the base64 payload into memory, decode it, and push it to Redis. If 10 users do this at the exact same millisecond, the Go API will require ~200MB of transient RAM. This could trigger an OOM kill if the container hits its 150MB limit. The queue worker will retry upon reboot."""
pdf.chapter_title('5. Edge Cases & Constraints')
pdf.chapter_body(body_5)

pdf.output('c:/project/Mail as a service/backend/MaaS_Resource_Report.pdf')
print("PDF generated successfully.")
