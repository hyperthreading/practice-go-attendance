import requests


def run_process(cmd):
    import subprocess

    process = subprocess.Popen(cmd, shell=True)
    return process


def wait_process(process):
    process.wait()
    if process.returncode != 0:
        raise Exception(f"Process failed with return code {process.returncode}")


def setup(context):
    server_process = run_process(
        "docker compose -f ./deploy/docker-compose.yaml up --build --wait -d"
    )
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
        test_fail_leave_before_attend(context)
        reset_time()
        test_success_attend_and_leave(context)
        reset_time()
    finally:
        teardown(context)


def test_health_check(context):
    response = requests.get(context.get("server_url", "") + "/health")
    assert response.status_code == 200
    assert response.json() == {"message": "ok"}


dummy_payload = {
    "token": "test-token",
    "team_id": "test-team-id",
    "team_domain": "test-team-domain",
    "channel_id": "test-channel-id",
    "channel_name": "test-channel-name",
    "user_id": "test-user-id",
    "user_name": "test-user-name",
    "command": "/attend",
    "text": "",
    "response_url": "test-response-url",
    "trigger_id": "test-trigger-id",
}


def fix_time(context, timeInString):
    data = {"time": timeInString}
    response = requests.post(
        context.get("server_url", "") + "/test/fix-time", json=data
    )
    print(response.json())
    assert response.status_code == 200


def reset_time(context):
    response = requests.post(context.get("server_url", "") + "/test/reset-time")
    assert response.status_code == 200


def attend(context):
    data = dummy_payload.copy()
    data["text"] = "attend"
    response = requests.post(
        context.get("server_url", "") + "/command/attend", data=data
    )
    return response


def leave(context):
    data = dummy_payload.copy()
    data["text"] = "leave"
    response = requests.post(
        context.get("server_url", "") + "/command/attend", data=data
    )
    return response


def test_fail_leave_before_attend(context):
    response = leave(context)
    assert response.status_code == 422
    assert response.json()["code"] == "not_attended_yet"


def test_success_attend_and_leave(context):
    fix_time(context, "2021-01-01T09:00:00Z")

    response = attend(context)
    assert response.status_code == 200

    fix_time(context, "2021-01-01T18:00:00Z")
    response_leave = leave(context)
    assert response_leave.status_code == 200


if __name__ == "__main__":
    test()
