package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	mimaDB "github.com/ahui2016/mima-go/db"
	"github.com/atotto/clipboard"
	"log"
	"net/http"
	"strings"
	"time"
)

type (
	httpRW  = http.ResponseWriter
	httpReq = *http.Request
	httpHF  = http.HandlerFunc
)

func main() {
	// 有 checkState 中间件的, 在 checkState 里对数据库加锁;
	// 没有 checkState 的, 要注意各自加锁.
	http.HandleFunc("/create-account", noCache(createAccount))
	http.HandleFunc("/change-password/", noCache(changePassword))
	http.HandleFunc("/login", noCache(loginHandler))
	http.HandleFunc("/logout", noCache(logoutHandler))
	http.HandleFunc("/home/", homeHandler)
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/index/", noCache(checkState(indexHandler)))
	http.HandleFunc("/search/", noCache(checkState(searchHandler)))
	http.HandleFunc("/add/", noCache(checkState(addPage)))
	http.HandleFunc("/api/add", checkLogin(addHandler))
	http.HandleFunc("/delete/", noCache(checkState(deleteHandler)))
	http.HandleFunc("/recyclebin/", noCache(checkState(recyclebin)))
	http.HandleFunc("/undelete/", noCache(checkState(undeleteHandler)))
	http.HandleFunc("/delete-forever/", noCache(checkState(deleteForever)))
	http.HandleFunc("/delete-tarballs/", noCache(deleteTarballs))
	http.HandleFunc("/edit/", noCache(checkState(editPage)))
	http.HandleFunc("/setup-ibm", noCache(setupIBM))
	http.HandleFunc("/recover-from-ibm/", noCache(recoverFromIBM))
	http.HandleFunc("/setup-cloud", noCache(setupIBM))
	http.HandleFunc("/backup-to-cloud/", noCache(checkState(backupToCloud)))
	http.HandleFunc("/backup-to-cloud-loading/", noCache(backupToCloudLoading))
	http.HandleFunc("/api/edit", checkLogin(editHandler))
	http.HandleFunc("/api/new-password", newPassword)
	http.HandleFunc("/api/delete-history", checkState(deleteHistory))
	http.HandleFunc("/api/copy-password", copyInBackground(copyPassword))
	http.HandleFunc("/api/copy-username", copyInBackground(copyUsername))
	http.HandleFunc("/api/count-tarballs", countTarballs)

	flag.Parse()
	addr := getAddr()
	term := getTerm()
	db.ValidTerm = time.Minute * time.Duration(term)
	fmt.Println(addr, "time limit:", term, "minutes")
	sessionManager = NewSessionManager(time.Hour * 12) // 默认 session 有效期为 12 小时
	log.Fatal(http.ListenAndServe(addr, nil))
}

func createAccount(w httpRW, r httpReq) {
	if !isLoggedOut(r) || !db.FileNotExist() {
		err := &Feedback{Err: errors.New("已存在账号, 不可重复创建")}
		checkErr(w, templates.ExecuteTemplate(w, "create-account", err))
		return
	}
	if r.Method != http.MethodPost {
		checkErr(w, templates.ExecuteTemplate(w, "create-account", nil))
		return
	}
	password := r.FormValue("password")
	if password == "" {
		err := &Feedback{Err: errors.New("密码不能为空")}
		checkErr(w, templates.ExecuteTemplate(w, "create-account", err))
		return
	}
	key := sha256.Sum256([]byte(password))
	if err := db.Init(&key); err != nil {
		checkErr(w, templates.ExecuteTemplate(w, "create-account", &Feedback{Err: err}))
		return
	}
	logout(w)
	info := &Feedback{Info: errors.New("成功创建新账号, 请登入")}
	checkErr(w, templates.ExecuteTemplate(w, "login", info))
}

func changePassword(w httpRW, r httpReq) {
	if isLoggedOut(r) || db.FileNotExist() {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if r.Method != http.MethodPost {
		checkErr(w, templates.ExecuteTemplate(w, "change-password", nil))
		return
	}
	oldPwd := r.FormValue("old-pwd")
	key := sha256.Sum256([]byte(oldPwd))
	if !db.EqualToUserKey(key) {
		err := &Feedback{Err: errors.New("当前密码错误, 为了提高安全性必须输入正确的当前密码")}
		checkErr(w, templates.ExecuteTemplate(w, "change-password", err))
		return
	}
	newPwd := r.FormValue("new-pwd")
	if err := db.ChangeUserKey(newPwd); err != nil {
		checkErr(w, templates.ExecuteTemplate(w, "change-password", &Feedback{Err: err}))
		return
	}
	logout(w)
	info := &Feedback{Info: errors.New("密码修改成功, 请使用新密码登入")}
	checkErr(w, templates.ExecuteTemplate(w, "login", info))
}

func loginHandler(w httpRW, r httpReq) {
	db.Lock()
	defer db.Unlock()
	if db.FileNotExist() {
		// 数据库不存在, 需要创建新账号.
		checkErr(w, templates.ExecuteTemplate(w, "create-account", nil))
		return
	}
	if !isLoggedOut(r) {
		err := &Feedback{Err: errors.New("已登入, 不可重复登入")}
		checkErr(w, templates.ExecuteTemplate(w, "login", err))
		return
	}
	if r.Method != http.MethodPost {
		checkErr(w, templates.ExecuteTemplate(w, "login", nil))
		return
	}
	password := r.FormValue("password")
	key := sha256.Sum256([]byte(password))
	if db.IsNotInit() {
		db.Reset()
		if _, err := db.Rebuild(&key); err != nil {
			logout(w)
			checkErr(w, templates.ExecuteTemplate(w, "login", &Feedback{Err: err}))
			return
		}
		// 必须更新时间, 这是容易忽略出错的地方.
		// 如果不更新时间, 会出现 "未登入, 已超时" 的错误.
		db.StartedAt = time.Now()
	}
	if key != db.UserKey() {
		err := errors.New("密码错误")
		checkErr(w, templates.ExecuteTemplate(w, "login", &Feedback{Err: err}))
		return
	}
	sessionManager.Add(w, mimaDB.NewID())

	log.Println("Logged in: 已登入")
	http.Redirect(w, r, "/home/", http.StatusFound)
}

func logoutHandler(w httpRW, _ httpReq) {
	logout(w)
	info := &Feedback{Info: errors.New("已登出, 请重新登入")}
	checkErr(w, templates.ExecuteTemplate(w, "login", info))
}

func setupIBM(w httpRW, r httpReq) {
	if r.Method != http.MethodPost {
		checkErr(w, templates.ExecuteTemplate(w, "setup-ibm", nil))
		return
	}
	settings, err := getSettings(r, false)
	if err != nil {
		checkErrForSetupIBM(w, err.Error(), &settings)
		return
	}
	if isLoggedOut(r) {
		checkErrForSetupIBM(w, "未登入(或已登出)", &settings)
		return
	}
	// 执行一次备份 (其中包括备份后下载回来检查数据是否一致)
	cos = newCOSFromSettings(&settings)
	if err := doBackup(); err != nil {
		checkErrForSetupIBM(w, err.Error(), &settings)
		return
	}
	// 更新设置 (持久化)
	if err := updateSettings(&settings); err != nil {
		checkErrForSetupIBM(w, err.Error(), &settings)
		return
	}
	// 显示成功信息
	cloudInfo := getCloudInfo()
	cloudInfo.Info = "云备份设置成功! 并且已备份一次."
	checkErr(w, templates.ExecuteTemplate(w, "backup-to-cloud", cloudInfo))
}

func recoverFromIBM(w httpRW, r httpReq) {
	if r.Method != http.MethodPost {
		checkErr(w, templates.ExecuteTemplate(w, "recover-from-ibm", nil))
		return
	}
	settings, err := getSettings(r, true)
	if err != nil {
		checkErrForRecoverFromIBM(w, err.Error(), &settings)
		return
	}
	if !db.FileNotExist() {
		checkErrForRecoverFromIBM(w, "在已存在数据库文件的情况下, 不可从云端恢复数据到本地", &settings)
		return
	}
	cos = newCOSFromSettings(&settings)
	data, err := cos.GetObjectBody(settings.ObjectName)
	if err != nil {
		checkErrForRecoverFromIBM(w, err.Error(), &settings)
		return
	}
	//noinspection GoUnhandledErrorResult
	defer data.Close()

	// 生成本地数据库文件, 其中 settings 已更新 (采用新的 prefix).
	settings64, err := settings.Encode()
	if err != nil {
		checkErrForRecoverFromIBM(w, err.Error(), &settings)
		return
	}
	if err := db.WriteDBFileFromReader(data, r.FormValue("password"), settings64); err != nil {
		checkErrForRecoverFromIBM(w, err.Error(), &settings)
		return
	}

	// 由于 prefix 已发生变化, 必须上传一次让云端也 "知道" 这个新的 prefix.
	// "知道" 是指执行 getCloudInfo() 时能顺利获取云端信息.
	if err := doBackup(); err != nil {
		checkErrForRecoverFromIBM(w, err.Error(), &settings)
		return
	}

	cloudInfo := getCloudInfo()
	cloudInfo.Info = "从云端下载数据库成功. 注意已自动生成新的 Object Name"
	checkErr(w, templates.ExecuteTemplate(w, "backup-to-cloud", cloudInfo))
}

func getSettings(r httpReq, needObjName bool) (settings Settings, err error) {
	var prefix string
	prefix = mimaDB.NewID()
	settings = Settings{
		ApiKey:            strings.TrimSpace(r.FormValue("apiKey")),
		ServiceInstanceID: strings.TrimSpace(r.FormValue("serviceInstanceID")),
		ServiceEndpoint:   strings.TrimSpace(r.FormValue("serviceEndpoint")),
		BucketLocation:    strings.TrimSpace(r.FormValue("bucketLocation")),
		BucketName:        strings.TrimSpace(r.FormValue("bucketName")),
		ObjKeyPrefix:      prefix,
		ObjectName:        strings.TrimSpace(r.FormValue("objectName")),
	}
	if !needObjName {
		settings.ObjectName = "不需要 Object Name, 因此乱填一些数据进来确保其不是空字符串"
	}
	if settings.ApiKey == "" || settings.ServiceInstanceID == "" || settings.ServiceEndpoint == "" ||
		settings.BucketLocation == "" || settings.BucketName == "" || settings.ObjectName == "" {
		err = errors.New("有漏填项目: 每个框都必须填写正确内容")
	}
	return
}

func checkErrForSetupIBM(w httpRW, errMsg string, settings *Settings) {
	settings.ErrMsg = errMsg
	checkErr(w, templates.ExecuteTemplate(w, "setup-ibm", settings))
}

func checkErrForRecoverFromIBM(w httpRW, errMsg string, settings *Settings) {
	settings.ErrMsg = errMsg
	checkErr(w, templates.ExecuteTemplate(w, "recover-from-ibm", settings))
}

func checkErrForBackupToCloud(w httpRW, errMsg string) {
	err := CloudInfo{Err: errMsg}
	checkErr(w, templates.ExecuteTemplate(w, "backup-to-cloud", &err))
}

func backupToCloudLoading(w httpRW, _ httpReq) {
	checkErr(w, templates.ExecuteTemplate(w, "backup-to-cloud-loading", nil))
}

func backupToCloud(w httpRW, r httpReq) {
	if !db.HasSettings() {
		http.Redirect(w, r, "/setup-cloud", http.StatusFound)
		return
	}
	if cos == nil {
		if err := makeCOS(db.GetSettings()); err != nil {
			checkErrForBackupToCloud(w, err.Error())
			return
		}
	}
	if r.Method != http.MethodPost {
		cloudInfo := getCloudInfo()
		checkErr(w, templates.ExecuteTemplate(w, "backup-to-cloud", cloudInfo))
		return
	}
	// 执行备份
	if err := doBackup(); err != nil {
		checkErrForBackupToCloud(w, err.Error())
		return
	}
	// 显示成功信息
	cloudInfo := getCloudInfo()
	cloudInfo.Info = "云备份成功! 最新的云端备份信息如下所示:"
	checkErr(w, templates.ExecuteTemplate(w, "backup-to-cloud", cloudInfo))
}

func getCloudInfo() *CloudInfo {
	lastModified, err := cos.GetLastModified(cos.MakeObjKey(DBName))
	if err != nil {
		return &CloudInfo{Err: err.Error()}
	}
	return &CloudInfo{
		CloudServiceName: "IBM Cloud Object Storage",
		BucketName:       cos.BucketName,
		ObjectName:       cos.MakeObjKey(DBName),
		LastModified:     lastModified.Local().Format(mimaDB.DateTimeFormat),
	}
}

func doBackup() error {
	// 尝试备份到云端
	buf, err := db.ReadMimaTable()
	if err != nil {
		return err
	}
	if _, err := cos.Upload(DBName, &buf); err != nil {
		return err
	}
	// 尝试从云端获取数据
	data, err := cos.GetObjectBody(cos.MakeObjKey(DBName))
	if err != nil {
		return err
	}
	//noinspection GoUnhandledErrorResult
	defer data.Close()
	// 检查从云端获取回来的数据与内存数据库是否一致 (只检查 UpdatedAt)
	// TODO: 更详细地检查
	return db.EqualByUpdatedAt(data)
}

func homeHandler(w httpRW, r httpReq) {
	switch r.URL.Path {
	case "/":
		fallthrough
	case "/home/":
		http.Redirect(w, r, "/search/", http.StatusFound)
	default:
		http.NotFound(w, r)
	}
}

func indexHandler(w httpRW, _ httpReq) {
	checkErr(w, templates.ExecuteTemplate(w, "index", db.All()))
}

func searchHandler(w httpRW, r httpReq) {
	if r.Method != http.MethodPost {
		checkErr(w, templates.ExecuteTemplate(w, "search", nil))
		return
	}
	alias := strings.TrimSpace(r.FormValue("alias"))
	if alias == "" {
		result := &SearchResult{Info: errors.New(
			"不可搜索空字符串, 请输入完整的别名, 本程序只能精确搜索, 区分大小写"),
		}
		checkErr(w, templates.ExecuteTemplate(w, "search", result))
		return
	}
	forms := db.GetFormsByAlias(alias)
	result := &SearchResult{SearchText: alias, Forms: forms}
	if forms == nil {
		result.Err = fmt.Errorf("NotFound: 找不到 alias: %s 的记录", alias)
	}
	checkErr(w, templates.ExecuteTemplate(w, "search", result))
}

func recyclebin(w httpRW, _ httpReq) {
	checkErr(w, templates.ExecuteTemplate(w, "recyclebin", db.DeletedMimas()))
}

func addPage(w httpRW, _ httpReq) {
	checkErr(w, templates.ExecuteTemplate(w, "add", nil))
}

func addHandler(w httpRW, r httpReq) {
	form := &MimaForm{
		Title:    strings.TrimSpace(r.FormValue("Title")),
		Username: strings.TrimSpace(r.FormValue("Username")),
		Password: r.FormValue("Password"),
		Notes:    strings.TrimSpace(r.FormValue("Notes")),
	}
	mima, err := mimaDB.NewMimaFromForm(form)
	if err == nil {
		err = db.Add(mima)
	}
	if err != nil {
		form.Err = err
		checkErr(w, templates.ExecuteTemplate(w, "add", form))
		return
	}
	form.ID = mima.ID
	result := &SearchResult{Forms: []*MimaForm{form}}
	checkErr(w, templates.ExecuteTemplate(w, "search", result))
}

func editPage(w httpRW, r httpReq) {
	form := new(MimaForm)
	id, ok := getAndCheckID(w, r, "edit", form)
	if !ok {
		return
	}
	form = db.GetFormByID(id)
	if form.IsDeleted() {
		form = &MimaForm{Err: errMimaDeleted}
	}
	checkErr(w, templates.ExecuteTemplate(w, "edit", form))
}

func editHandler(w httpRW, r httpReq) {
	form := new(MimaForm)
	id, ok := getAndCheckID(w, r, "edit", form)
	if !ok {
		return
	}
	form = &MimaForm{
		ID:       id,
		Title:    strings.TrimSpace(r.FormValue("Title")),
		Alias:    strings.TrimSpace(r.FormValue("Alias")),
		Username: strings.TrimSpace(r.FormValue("Username")),
		Password: r.FormValue("Password"),
		Notes:    strings.TrimSpace(r.FormValue("Notes")),
		History:  form.History,
	}
	if form.Err = db.Update(form); form.Err != nil {
		checkErr(w, templates.ExecuteTemplate(w, "edit", form))
		return
	}
	result := &SearchResult{Forms: []*MimaForm{form}}
	checkErr(w, templates.ExecuteTemplate(w, "search", result))
}

func getAndCheckID(w httpRW, r httpReq, tmpl string, form *MimaForm) (id string, ok bool) {
	if id = strings.TrimSpace(r.FormValue("id")); id == "" {
		form.Err = fmt.Errorf("id 不可为空")
		checkErr(w, templates.ExecuteTemplate(w, tmpl, form))
		return
	}
	return id, true
}

func deleteHandler(w httpRW, r httpReq) {
	form := new(MimaForm)
	id, ok := getAndCheckID(w, r, "delete", form)
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		form = db.GetFormByID(id).HideSecrets()
		if form.IsDeleted() {
			form = &MimaForm{Err: errMimaDeleted}
		}
		checkErr(w, templates.ExecuteTemplate(w, "delete", form))
		return
	}
	if err := db.TrashByID(id); err != nil {
		form.Err = err
		checkErr(w, templates.ExecuteTemplate(w, "delete", form))
		return
	}
	http.Redirect(w, r, "/home/", http.StatusFound)
}

func deleteTarballs(w httpRW, r httpReq) {
	fb := new(Feedback)
	fragFiles, err := db.GetTarballPaths()
	if err != nil {
		fb.Err = err
	}
	n := len(fragFiles)
	if r.Method != http.MethodPost {
		fb.Number = n
		checkErr(w, templates.ExecuteTemplate(w, "delete-tarballs", fb))
		return
	}
	if n > 10 {
		if err := mimaDB.DeleteFiles(fragFiles[:n-10]); err != nil {
			fb.Err = err
		}
	}
	checkErr(w, templates.ExecuteTemplate(w, "delete-tarballs", fb))
}

func undeleteHandler(w httpRW, r httpReq) {
	form := new(MimaForm)
	id, ok := getAndCheckID(w, r, "undelete", form)
	if !ok {
		return
	}
	form = db.GetFormByID(id)
	if !form.IsDeleted() {
		form := &MimaForm{Err: errors.New("回收站中找不到此记录: " + id)}
		checkErr(w, templates.ExecuteTemplate(w, "undelete", form))
		return
	}
	if r.Method != http.MethodPost {
		checkErr(w, templates.ExecuteTemplate(w, "undelete", form))
		return
	}
	if err := db.UnDeleteByID(id); err != nil {
		form = &MimaForm{Err: err}
		checkErr(w, templates.ExecuteTemplate(w, "undelete", form))
		return
	}
	result := &SearchResult{Forms: []*MimaForm{form}}
	checkErr(w, templates.ExecuteTemplate(w, "search", result))
}

func deleteForever(w httpRW, r httpReq) {
	form := new(MimaForm)
	id, ok := getAndCheckID(w, r, "delete-forever", form)
	if !ok {
		return
	}
	form = db.GetFormByID(id)
	if !form.IsDeleted() {
		form := &MimaForm{Err: errors.New("回收站中找不到此记录: " + id)}
		checkErr(w, templates.ExecuteTemplate(w, "delete-forever", form))
		return
	}
	if r.Method != http.MethodPost {
		checkErr(w, templates.ExecuteTemplate(w, "delete-forever", form))
		return
	}
	checkErr(w, db.DeleteForeverByID(id))
	http.Redirect(w, r, "/recyclebin/", http.StatusFound)
}

func deleteHistory(w httpRW, r httpReq) {
	id := strings.TrimSpace(r.FormValue("id"))
	if id == "" {
		http.Error(w, "id 不可为空", http.StatusNotAcceptable)
		return
	}
	datetime := strings.TrimSpace(r.FormValue("datetime"))
	if len(datetime) < len(mimaDB.DateTimeFormat) {
		http.Error(w, fmt.Sprintf("格式错误: %s", datetime), http.StatusConflict)
		return
	}
	if err := db.DeleteHistoryItem(id, datetime); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
	}
}

func countTarballs(w httpRW, _ httpReq) {
	fragFiles, err := db.GetTarballPaths()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	if len(fragFiles) <= 10 {
		http.Error(w, "不超过 10 个备份文件, 不需要删除.", http.StatusNotAcceptable)
	}
}

/*
func writeJSON(w httpRW, msg string, status int) {
	w.WriteHeader(status)
	checkErr(w, json.NewEncoder(w).Encode(msg))
}
*/

func newPassword(w httpRW, _ httpReq) {
	pwBytes := make([]byte, passwordSize)
	if _, err := rand.Read(pwBytes); err != nil {
		_, _ = fmt.Fprint(w, err)
	}
	pw := base64.RawURLEncoding.EncodeToString(pwBytes)[:passwordSize]
	_, _ = fmt.Fprint(w, pw)
}

func copyPassword(mima *Mima) {
	_ = copyToClipboard(mima.Password)
}

func copyUsername(mima *Mima) {
	_ = copyToClipboard(mima.Username)
}

func checkErr(w httpRW, err error) {
	if err != nil {
		log.Println(err)
		_, _ = fmt.Fprintf(w, "%v", err)
	}
}

func logout(w httpRW) {
	db.Reset()
	sessionManager.DeleteSID(w)
	log.Println("Logged out: 已登出")
}

func isLoggedOut(r httpReq) bool {
	return db.IsNotInit() || !sessionManager.Check(r)
}

// 复制到剪贴板, 并在一定时间后清空剪贴板.
func copyToClipboard(s string) (err error) {
	if err = clipboard.WriteAll(s); err != nil {
		return
	}

	// 三十秒后自动清空剪贴板.
	<-time.After(time.Second * 30)

	var text string
	if text, err = clipboard.ReadAll(); err != nil {
		return
	}
	if text == s {
		return clipboard.WriteAll("")
	}
	return nil
}
