import subprocess
import requests
import os
import time

def run_process(cmd):

    process = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, shell=True)
    return process


def wait_process(process: subprocess.Popen, raise_on_error=True):
    out, err = process.communicate()
    if raise_on_error and process.returncode != 0:
        raise Exception(f"Process failed with return code {process.returncode}")
    return out, err


def run_watch_process(context):
    # check if the container is already running
    watch_lock_path = os.path.join(os.path.dirname(__file__), "watch.lock")
    running = False
    with open(watch_lock_path, "a+") as f:
        f.seek(0)
        container_id = f.read()

    container_id = container_id.strip()
    if container_id:
        cmd = f"docker inspect {container_id}"
        process = run_process(cmd)
        wait_process(process, raise_on_error=False)
        running = process.returncode == 0
    
    if not running:
    
        cmd = "docker build -t go-builder -f ./build/test/builder.Dockerfile ."
        wait_process(run_process(cmd))

        cmd = "docker run -q -d --rm --name go-builder -v $(pwd):/app go-builder"
        container_id = wait_process(run_process(cmd))[0].decode("utf-8").strip()
        
        with open(watch_lock_path, "w") as f:
            f.write(container_id)
    
    cmd = f"docker logs -f {container_id}"
    process = run_process(cmd)
    context["build_log_process"] = process


def setup(context):
    run_watch_process(context)
    server_process = run_process(
        "docker compose -f ./deploy/test/docker-compose.yaml up --wait -d"
    )
    wait_process(server_process)
    log_process = run_process("docker compose -f ./deploy/docker-compose.yaml logs -f")
    context["log_process"] = log_process


def teardown(context):
    cmd = run_process("docker compose -f ./deploy/test/docker-compose.yaml down")
    wait_process(cmd)
    context["log_process"].terminate()
    context["build_log_process"].terminate()


def test():
    context = {}
    setup(context)
    context["server_url"] = "http://localhost:8080"
    try:
        test_health_check(context)
        test_fail_leave_before_attend(context)
        reset_time(context)
        test_success_attend_and_leave(context)
        reset_time(context)
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
