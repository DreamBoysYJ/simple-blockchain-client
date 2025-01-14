# Simple-Blockchain-Client (가제) v1

## 📖 프로젝트 개요
이 프로젝트는 이더리움 기반으로 간단한 블록체인 클라이언트를 처음부터 구현했습니다.  
***로컬환경에서 P2P 통신부터 트랜잭션과 블록의 검증, 전파, 저장, 실행***을 포함합니다.  

## 📖 프로젝트 목적
- **Go 언어 학습**  
  고루틴과 채널 등 Go의 다양한 기능을 프로젝트에 활용하며, 언어의 깊은 학습을 하고자 합니다.

- **블록체인 코어 기술에 대한 심층 이해**  
  블록체인 핵심 기술을 직접 구현하면서, 이론적으로 알던 개념들을 코드로 구체화하며 각 기술의 중요성, 필요성을 느끼고자 합니다.  
  예를 들어, 저는 데이터를 JSON으로 직렬화 했습니다. 그러나 구현 과정에서 타입의 명확성 부족, 비효율적인 크기를 경험하며 RLP의 효율성을 체감했습니다.

- **분산 시스템 프로그램 및 객체 지향 설계 학습**  
  규모있는 프로그램을 체계적으로 설계하고 구현하는 능력을 키우고 싶었습니다.  
  프로젝트 규모가 커짐에 따라, 객체 지향 설계(OOP)를 적용해 설계를 체계화해야 코드의 유지보수와 확장이 용이함을 깨달았습니다.

---

## 🛠 사용 가이드

### 사전 준비 사항
아래 도구들을 사전에 설치하고 준비하셔야 합니다!:

1. **Go**  
   다운로드 및 설치: [Go 공식 사이트](https://go.dev/dl/)

2. **Postman API Document**  
   준비: [Postman API 문서](https://documenter.getpostman.com/view/25348775/2sAYQWLZZ9#8e9b9e8f-e009-4a6e-9e87-fd65f566420a)  
   Postman을 통해 블록체인 노드에 트랜잭션 전송, 블록 조회 등이 가능합니다.

---

### 설치 및 실행

1. 프로젝트를 클론한 후 프로젝트 디렉토리로 이동하세요:
    ```bash
    git clone https://github.com/DreamBoysYJ/simple-blockchain-client.git
    cd simple-blockchain-client
    ```

2. **글로벌 명령어를 설정하세요 (옵션)**:  
   프로젝트의 주요 명령어를 실행할 수 있도록 입력하세요.
    ```bash
    make all
    ```
   프로젝트를 빌드한 후, 실행파일을 `/usr/local/bin` 디렉토리에 설치하여 시스템 전역에서 사용할 수 있습니다.

3. **`bootnode(bootstrap node)`를 실행하세요**:
    ```bash
    go run . -nodeID=boot -mode=bootnode
    
    # 2번을 진행하셨다면:
    
    simple-blockchain-client -nodeID=boot -mode=bootnode
    ```
    bootnode는 UDP 서버로, 노드가 처음 실행될 때 연결하여 다른 노드의 정보를 수신하기 위한 역할을 합니다.  
    `Node Discovery`를 참고하여 간단한 프로토콜을 설계했고, bootnode와 통신한 노드들의 정보를 전달받습니다.

4. **`Fullnode`를 서로 다른 터미널에서 최소 3개 이상 실행하세요**:
    ```bash
    go run . -nodeID=node1 -mode=fullnode -port=30301 -rpcport=8081
    go run . -nodeID=node2 -mode=fullnode -port=30302 -rpcport=8082
    go run . -nodeID=node3 -mode=fullnode -port=30303 -rpcport=8083
    
    ### 2번을 진행하셨다면:

    simple-blockchain-client -nodeID=node1 -mode=fullnode -port=30301 -rpcport=8081
    simple-blockchain-client -nodeID=node2 -mode=fullnode -port=30302 -rpcport=8082
    simple-blockchain-client -nodeID=node3 -mode=fullnode -port=30303 -rpcport=8083
    ```

**Fullnode 주요 역할**:
| 역할               | 설명                                                         |
|--------------------|--------------------------------------------------------------|
| **P2P 네트워크**   | UDP, TCP 서버 실행 및 메시지 브로드캐스팅                     |
| **블록 생성**      | 주기적으로 멤풀을 확인해 블록 생성 시도                       |
| **JSON-RPC 서버**  | 외부와의 통신 제공                                           |
| **초기화 작업**    | 계정, LevelDB, 제네시스 블록 설정                            |

자세한 내용은 [Docs/Fullnode.md](#), [Docs/Blockchain.md](#)를 참고하세요.

---

### 플래그 설명

| 플래그      | 설명                                                                                     | 기본값         |
|-------------|------------------------------------------------------------------------------------------|----------------|
| `port`      | `Node Discovery`를 위한 UDP 서버 포트이면서, P2P 통신을 위한 TCP 서버 포트               | 30303          |
| `rpcport`   | 외부 브라우저나 DApp과 통신하기 위한 JSON-RPC 서버 포트                                  | 8080           |
| `nodeID`    | 각 노드를 구분하기 위한 식별자. 데이터베이스 경로(`dbPath`) 설정에 사용되며, 로컬 테스트용 | `default`      |
| `mode`      | 노드의 역할을 지정 (`bootnode` 또는 `fullnode`). 미입력 시 기본적으로 `fullnode`로 설정됨 | `fullnode`     |

![image](https://github.com/user-attachments/assets/5157266f-d262-4353-aa5c-ed9f64853e53)
위와 같이 노드를 위한 계정 생성, 제네시스 블록 생성, 노드 연결을 통한 P2P 구축을 진행합니다.

---

### 5. Postman으로 트랜잭션 테스트하기

1. **상태 확인**: 우선 `getLastBlock`, `getBlockNumber`, `getAccountInfo`를 호출하여 현재 상태를 확인해주세요.
2. **트랜잭션 생성**: tx1~tx10, `SendTransaction`을 호출하여 서명된 트랜잭션을 전송하세요.
   - `SendRawTransaction`을 통해 서명된 값을 미리 세팅해두었습니다.
   - 메시지는 `from`, `to`, `value`, `nonce`를 붙인 값을 사용했습니다.
   - `from`의 주소는 현재는 해당 주소만 사용할 수 있습니다. metamask 같은 툴을 사용할 수 없기 때문에 해당 주소의 개인키는 하드코딩, value는 제네시스 블록 Miner로 이 주소를 세팅해 10000이 잔고로 있습니다.
   - tx1에서 tx10까지 어떤 순서로 실행해도 괜찮습니다. 멤풀에서 계정 별로 논스를 기준으로 정렬하기 때문입니다.
   - 그러나 현재는 주소 하나의 트랜잭션들만 멤풀에 담기기 떄문에, tx1~tx5까지 전송을 해야만 블록을 생성할 것입니다. (멤풀에서 주소마다 논스 순으로 Round Robin으로 트랜잭션을 추출해 블록을 생성합니다)
   - 각 tx은 유효성 검증 후 피어에게 전파되나, 무한 전파를 막기 위해 이미 멤풀에 있는 중복 tx일 경우 drop합니다.




### 6. 터미널로 블록 생성 확인하기

1. **상태 확인**: 우선 `getLastBlock`, `getBlockNumber`, `getAccountInfo`를 호출하여 현재 상태를 확인해주세요.
2. **트랜잭션 생성**: tx1~tx10, `SendTransaction`을 호출하여 서명된 트랜잭션을 전송하세요.
   - `SendRawTransaction`을 통해 서명된 값을 미리 세팅해두었습니다.
   - 메시지는 `from`, `to`, `value`, `nonce`를 붙인 값을 사용했습니다.
   - `from`의 주소는 현재는 해당 주소만 사용할 수 있습니다. metamask 같은 툴을 사용할 수 없기 때문에 해당 주소의 개인키는 하드코딩, value는 제네시스 블록 Miner로 이 주소를 세팅해 10000이 잔고로 있습니다.
   - tx1에서 tx10까지 어떤 순서로 실행해도 괜찮습니다. 멤풀에서 계정 별로 논스를 기준으로 정렬하기 때문입니다.
   - 그러나 현재는 주소 하나의 트랜잭션들만 멤풀에 담기기 떄문에, tx1~tx5까지 전송을 해야만 블록을 생성할 것입니다. (멤풀에서 주소마다 논스 순으로 Round Robin으로 트랜잭션을 추출해 블록을 생성합니다)
   - 각 tx은 유효성 검증 후 피어에게 전파되나, 무한 전파를 막기 위해 이미 멤풀에 있는 중복 tx일 경우 drop합니다.

   
