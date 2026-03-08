import requests
import time
import os
import threading
import json
import uuid
import subprocess

BASE_URL = "http://localhost:8000"

def run_cmd(cmd):
    return subprocess.check_output(cmd, shell=True).decode('utf-8')

print("=== Edge Case Testing Suite ===")

# 1. Register a test user
test_email = f"test_{uuid.uuid4().hex[:6]}@example.com"
test_password = "password123"

print(f"\n[*] Creating test user {test_email}...")
res = requests.post(f"{BASE_URL}/auth/register", json={"email": test_email, "password": test_password})
if res.status_code not in (200, 201):
    print("Failed to register:", res.text)
    
# 2. Login
print("[*] Logging in to get JWT...")
res = requests.post(f"{BASE_URL}/auth/login", json={"email": test_email, "password": test_password})
token = res.json().get('access_token')

headers = {"Authorization": f"Bearer {token}"}

# 3. Test API Load (100 rapid /mail/send requests)
print("\n[*] Testing Moderate Load (100 concurrent sends without attachments)")

def send_mail():
    try:
        requests.post(
            f"{BASE_URL}/mail/send",
            headers=headers,
            files={"to": (None, "recipient@example.com"), "subject": (None, "Load Test"), "body": (None, "Hello World")}
        )
    except Exception as e:
        pass

start_time = time.time()
threads = []
for _ in range(100):
    t = threading.Thread(target=send_mail)
    t.start()
    threads.append(t)

for t in threads:
    t.join()

print(f"[+] 100 requests finished in {time.time() - start_time:.2f} seconds")

# 4. Test Large Attachment (Memory limit check)
print("\n[*] Testing Large Attachment Edge Case (10MB streaming)")
# Generate a 10MB dummy file
with open("large_file.dat", "wb") as f:
    f.write(os.urandom(10 * 1024 * 1024))

try:
    with open("large_file.dat", "rb") as f:
        print("[*] Uploading 10MB file...")
        res = requests.post(
            f"{BASE_URL}/mail/send",
            headers=headers,
            files={
                "to": (None, "recipient@example.com"),
                "subject": (None, "10MB Attachment Test"),
                "body": (None, "Here is your large file"),
                "attachment1": ("large_file.dat", f, "application/octet-stream")
            }
        )
    print(f"[+] Large upload response: {res.status_code} - {res.text}")
except Exception as e:
    print(f"[-] Large upload failed: {e}")
finally:
    os.remove("large_file.dat")

print("\n=== Docker Stats (Check RAM limits) ===")
print(run_cmd("docker stats --no-stream --format \"table {{.Name}}\t{{.MemUsage}}\t{{.MemPerc}}\t{{.CPUPerc}}\""))
