# ADR002-Databse

## Status

Approved

## Decision Outcome

データベースとしてPostgreSQLを採用します｡
これは､TACOKUMOをデプロイするKubernetesクラスタの生存区間よりも､
アプリケーションのデータの方が長く存続することを想定しているためです｡
そのため､データベースはアプリケーションとは独立して管理される必要があります｡

一方で､開発初期段階としては､各データベースの情報はKubernetes上のカスタムリソースとほぼ同等の情報のみ持つため､
PostgreSQLを使わずに､シンプルにKubernetes APIだけを使います｡

