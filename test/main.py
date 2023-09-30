import requests
import time


def run_process(cmd):
    import subprocess

    process = subprocess.Popen(cmd, shell=True)
    return process

def wait_process(process):
    process.wait()
    if process.returncode != 0:
        raise Exception(f"Process failed with return code {process.returncode}")

def setup(context):
    server_process = run_process("docker compose -f ./deploy/docker-compose.yaml up --build --wait -d")
    wait_process(server_process)    
    log_process = run_process("docker compose -f ./deploy/docker-compose.yaml logs -f")
    context["log_process"] = log_process


def teardown(context):
    cmd = run_process("docker compose -f ./deploy/docker-compose.yaml down")
    wait_process(cmd)
    context["log_process"].terminate()


def test():
    context = {}
    setup(context)
    context["server_url"] = "http://localhost:8080"
    try:
        test_health_check(context)
        test_receive_slack_command(context)
    finally:
        teardown(context)


def test_health_check(context):
    response = requests.get(context.get("server_url", "") + "/health")
    assert response.status_code == 200
    assert response.json() == {"message": "ok"}


def test_receive_slack_command(context):
    response = requests.post(context.get("server_url", "") + "/command/attendance", json={
        "token": "test-token",
        "team_id": "test-team-id",
        "team_domain": "test-team-domain",
        "channel_id": "test-channel-id",
        "channel_name": "test-channel-name",
        "user_id": "test-user-id",
        "user_name": "test-user-name",
        "command": "/attendance",
        "text": "test-text",
        "response_url": "test-response-url",
        "trigger_id": "test-trigger-id",
    })
    assert response.status_code == 200


if __name__ == "__main__":
    test()
