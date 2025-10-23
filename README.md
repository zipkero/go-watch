## go-watch

HTTP 요청을 반복 실행하고 응답 시간을 측정하는 도구

### 빌드

```shell
go build -o go-watch.exe ./cmd
```

### 사용법

```shell
./go-watch.exe config.yaml
```

### 설정 예시

#### 기본 GET 요청

```yaml
url: https://api.example.com/users
method: GET
requests: 10
concurrency: 2
delay: 1
```

#### 헤더와 쿼리 파라미터

```yaml
url: https://api.example.com/search
method: GET
requests: 5
concurrency: 1
delay: 0

query_params:
  q: golang
  limit: "10"

headers:
  Authorization: Bearer your-token
  User-Agent: go-watch/1.0
```

#### POST 요청 (JSON)

```yaml
url: https://api.example.com/users
method: POST
requests: 3
concurrency: 1
delay: 1

headers:
  Content-Type: application/json

body_type: json
body:
  name: John
  email: john@example.com
```

#### POST 요청 (Form)

```yaml
url: https://api.example.com/login
method: POST
requests: 1
concurrency: 1
delay: 0

body_type: form
body:
  username: admin
  password: secret
```

#### Pre-request Script

요청 전에 JavaScript를 실행하여 동적으로 헤더를 생성할 수 있습니다.

```yaml
url: https://api.example.com/data
method: GET
requests: 5
concurrency: 1
delay: 1

pre_request_script: |
  let apiKey = "YOUR_API_KEY";
  let secret = "YOUR_SECRET";
  let timestamp = Math.round(Date.now() / 1000);
  let signature = sha512(apiKey + secret + timestamp);

  env.set("apiKey", apiKey);
  env.set("signature", signature);
  env.set("timestamp", timestamp);

headers:
  X-API-Key: "{{apiKey}}"
  X-Signature: "{{signature}}"
  X-Timestamp: "{{timestamp}}"
```

사용 가능한 함수:
- `env.set(key, value)`: 변수 저장
- `env.get(key)`: 변수 조회
- `sha256(text)`: SHA256 해시
- `sha512(text)`: SHA512 해시
- `Date.now()`: 현재 타임스탬프(밀리초)

#### 결과 저장

```yaml
url: https://api.example.com/status
method: GET
requests: 10
concurrency: 5
delay: 0

output_file: results.jsonl
save_response_body: true
```