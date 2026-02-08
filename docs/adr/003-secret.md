# ADR003-Secret

## Status

Approved

## Context

TACOKUMOで管理されるいくつかのアプリケーションでは､Slack BotにおけるAPI Tokenなど､機密情報を必要とするものがあります｡

TACOKUMOでこれらの機密情報をどのように管理できるようにするかを決定します｡

## Decision Outcome

TACOKUMOでは､最終的にKubernetes Secretとして保存されコンテナから参照されるようにします｡

一方で､TACOKUMOクラスタはKubernetesクラスタよりも長く生存するため､機密情報の源泉はPostgreSQLに保存します｡

データベースには､以下のようにして保存します｡
それぞれの機密情報には｢暗号化に使われた鍵｣が紐づけられます｡
これによって､ふたび機密情報を復号化するために必要な鍵を特定できます｡

鍵は定期的にローテーションされます｡
`secret_keys.created_at` の値によって､古い鍵を使っていることを特定できます｡

ユーザは任意のタイミングで､鍵をローテーションできます｡
鍵をローテーションした際には､すべての機密情報が新しい鍵で再暗号化されます｡

最小のスコープを保つために､secretに対し一つの鍵を紐づける設計とします｡

```sql
CREATE TABLE secrets (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    encrypted_data BYTEA NOT NULL,
    key_id INTEGER NOT NULL REFERENCES secret_keys(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE secret_keys (
    id SERIAL PRIMARY KEY,
    key BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### コンポーネント設計

`secret_keys` に対する権限を持つコンポーネントを独自に設計し､
portal apiからはRPCでアクセスする形にします｡
これにより､鍵の管理を専門に行うコンポーネントを将来的に分離しやすくなります｡

実装初期段階では､portal apiが直接`secret_keys`にアクセスします｡
