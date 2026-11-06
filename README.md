# nemokan

nemokan(네모칸)은 다이어그램과 화면 와이어프레임을 글자로 그리는 편집기다.
그린 것은 Markdown 으로 복사해 문서, 이슈, AI 에이전트에게 바로 넘기거나 SVG, PNG 로 내려받는다.

## 출처

- [MonoSketch](https://github.com/tuanchauict/MonoSketch) (Tuan Chau) 의
  [커밋 6eeb753](https://github.com/tuanchauict/MonoSketch/tree/6eeb753b436509e49507a66344be5c53c890e8f5) 을 바탕으로,
  Kotlin/JS 코드를 Go 로 옮겼다. 편집기는 브라우저에서 Go WebAssembly 로 돈다.
- 화면 배치와 UI 요소는 [Mockdown](https://github.com/bbssppllvv/Mockdown) 에서 영감을 받았다. Mockdown 의 코드는 들어 있지 않다.

## 기능

- 사각형, 글자, 선. 선은 도형에 붙어 있어서 도형을 옮겨도 따라온다.
- UI 요소 19가지 (버튼, 입력칸, 표, 탭, 모달, 체크박스 등)
- 그룹과 레이어 (숨기기, 잠그기, 이름 바꾸기, 순서 바꾸기)
- 한글처럼 폭이 넓은 글자는 두 칸을 차지해서, 글자로 내보내도 줄이 어긋나지 않는다.
- Markdown 복사, SVG, PNG 내려받기
- 문서는 먼저 브라우저에 저장되고, 로그인하면 서버와 맞춰져 다른 기기에서 열거나 링크로 공유할 수 있다.

## 실행

Go 와 Node.js 가 필요하다.

```sh
make run
```

<http://localhost:8080> 에서 연다. 기본 설정은 서버 DB 가 메모리라서, 서버를 다시 켜면 계정과 서버 문서가 사라진다.

첫 관리자 계정은 <http://localhost:8080/bootstrap> 에서 만든다. 이 페이지를 열면 서버 로그에 부트스트랩 토큰이 찍힌다.

## 개발

```sh
make test     # go fmt, go vet, go test
make dev-run  # 파일이 바뀌면 다시 빌드하고 서버를 다시 켬
```

설계 결정은 [docs/adr](docs/adr), 할 일은 [docs/tasks.md](docs/tasks.md) 에 있다.

## 라이선스

Apache License 2.0. [LICENSE](LICENSE) 와 [NOTICE](NOTICE) 를 본다.
