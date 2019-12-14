## Write to file

- The `Fprintln` function takes a `io.writer` as parameter and appends a new line.

## Secretbox

- 在 json_test.go 里对加密进行试验.
- 测试多条加密数据的文件读写（换行符的处理）

## 数据库操作符号

- 在 Mima 中增加一个数据库操作符号
  (不需要, 利用 UpdatedAt 来表示)

## 唯一性检查

- Nonce?
- 由于数据量少, 就用遍历检查.
- 以后数据量大, 可以增加一个 map 来提高寻找唯一主键的效率.
