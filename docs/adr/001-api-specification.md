# ADR001 API Specification

## Status

Approved

## Decision Outcome

portal-apiは､connectrpcを利用してスキーマ駆動で開発します｡
APIの方式としてはシンプルなHTTP/JSONを採用しますが､
connect-goを利用することで､protobufを利用した開発が可能となります｡

スキーマ定義ファイルは､ `tacokumo/api-spec` などの分割されたリポジトリではなく､
portal-apiリポジトリ内に配置します｡
これは､開発初期段階において､API仕様の変更が頻繁に発生することが予想されるためです｡

将来的にAPI仕様が安定した段階で､ `tacokumo/api-spec` のような分割されたリポジトリに
移行することを検討します｡
