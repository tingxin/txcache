package txcache

import (
"bufio"
"fmt"
"log"
"os"
"path/filepath"
"runtime"
"sync"
"time"
"github.com/go-openapi/errors"
)

const (
	CacheSTRING = iota
	CacheNUMBER
	CacheBOOL
	CacheFILE
	CacheUnKnown
	CacheObject
)

const (
	NeverExpired   = -1
	DefaultExpired = 10 * 60
	MinuteExpired  = 60
	HourExpired    = 60 * 60
	DayExpired     = HourExpired * 24
	WeekExpired    = DayExpired * 7
	MonthExpired   = WeekExpired * 4
	YearExpired    = MonthExpired * 12
)

const  (
	KeyNotFound = 1
	FetcherCanNotGetData = 2
	ValueExpired

)

type Object interface {
}

type CFetcher func(arguments ...Object) (value Object, result bool)

type CacheValue struct {
	value            Object
	fetcher          CFetcher
	fetcherArguments []Object
	valueTime        time.Time
	expireSeconds    float32
	valueType        int
	storedPath       string
	frequency        int
}

type CacheConfig struct {
	MemoryMaxSize int
	FileStorePath string
	dataFolder    string
}

type Cache struct {
	contents map[string]*CacheValue
	mutex    sync.RWMutex
	cfg      CacheConfig
}

func init() {
	__default = NewCache()
}

var __default *Cache

func NewCache() *Cache {

	localDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	defaultCfg := CacheConfig{FileStorePath: localDir, MemoryMaxSize: 512}
	return NewCacheWithConfig(defaultCfg)
}

func NewCacheWithConfig(cfg CacheConfig) *Cache {
	cache := Cache{contents: make(map[string]*CacheValue)}
	dataFolder, _ := CreateTargetDirs(cfg.FileStorePath, ".cache")
	cfg.dataFolder = dataFolder
	cache.cfg = cfg
	go cache.gc()
	return &cache
}

func Default() *Cache {
	return __default
}

func (sender *Cache) Set(key string, value Object, expireSeconds float32) {
	sender.mutex.Lock()
	valueType := CacheUnKnown
	switch value.(type) {
	case string:
		valueType = CacheSTRING
	case int:
		valueType = CacheNUMBER
	case int32:
		valueType = CacheNUMBER
	case int64:
		valueType = CacheNUMBER
	case bool:
		valueType = CacheBOOL
	}

	if cacheValue, checker := sender.contents[key]; checker {

		cacheValue.valueTime = time.Now()
		cacheValue.expireSeconds = expireSeconds
		cacheValue.fetcher = nil
		cacheValue.fetcherArguments = nil
		cacheValue.valueType = valueType
		cacheValue.value = value
		cacheValue.storedPath=""
		go sender.saveCacheItem(key, value, cacheValue)

	} else {

		var newCacheValue CacheValue = CacheValue{value: value, valueTime: time.Now(), fetcher: nil, expireSeconds: expireSeconds, fetcherArguments: nil}
		newCacheValue.valueType = valueType
		sender.contents[key] = &newCacheValue
		go sender.saveCacheItem(key, value, &newCacheValue)
	}
	sender.mutex.Unlock()
}

func (sender *Cache) SetWithFetcher(key string, fetcher CFetcher, expireSeconds float32, arguments ...Object) {
	sender.mutex.Lock()

	if cacheValue, checker := sender.contents[key]; checker {
		cacheValue.valueTime = time.Now()
		cacheValue.expireSeconds = expireSeconds
		cacheValue.fetcher = fetcher
		cacheValue.fetcherArguments = arguments
		cacheValue.value = nil

	} else {

		var newCacheValue CacheValue = CacheValue{value: nil, valueTime: time.Now(), fetcher: fetcher, expireSeconds: expireSeconds, fetcherArguments: arguments}
		sender.contents[key] = &newCacheValue
	}
	sender.mutex.Unlock()
}

func (sender *Cache) Get(key string) (result Object, err error) {

	if cacheItem, ok := sender.contents[key]; ok {
		timeExpired := cacheItem.expireSeconds > 0 && float32(time.Now().Sub(cacheItem.valueTime).Seconds()) > cacheItem.expireSeconds
		if timeExpired || cacheItem.value == nil {

			if cacheItem.fetcher != nil {
				if newValue, test := cacheItem.fetcher(cacheItem.fetcherArguments...); test {
					cacheItem.valueTime = time.Now()
					cacheItem.frequency += 1
					cacheItem.value = newValue
					result = newValue
					err = nil
					go sender.saveCacheItem(key, newValue, cacheItem)
				}else{
					err = errors.New(FetcherCanNotGetData, key + "'s fetcher can not get data")
				}
			} else {
				result = nil
				err = errors.New(ValueExpired, key +  "'s value expired")
			}

		} else {

			if len(cacheItem.storedPath) > 0 && cacheItem.frequency == 0 {
				if content, ok := ReadFile(cacheItem.storedPath); ok {
					result = string(content)
				}
			} else {
				result = cacheItem.value
			}
			cacheItem.frequency += 1
			err = nil
		}
	} else {
		result = nil
		err = errors.New(KeyNotFound, key + " not found")
	}
	return
}

func (sender *Cache) GetString(key string) (result string, err error) {
	result1, err1 := sender.Get(key)
	err = err1
	if err ==nil{
		if typedValue, isTypeValue := result1.(string); isTypeValue {
			result = typedValue
		}
	}
	return
}

func (sender *Cache) GetInt(key string) (result int, err error) {
	result1, err1 := sender.Get(key)
	err = err1
	if err ==nil{
		if typedValue, isTypeValue := result1.(int); isTypeValue {
			result = typedValue
		}
	}
	return
}

func (sender *Cache) GetFloat64(key string) (result float64, err error) {

	result1, err1 := sender.Get(key)
	err = err1
	if err ==nil{
		if typedValue, isTypeValue := result1.(float64); isTypeValue {
			result = typedValue
		}
	}
	return
}

func (sender *Cache) GetBool(key string) (result bool, err error) {

	result1, err1 := sender.Get(key)
	err = err1
	if err ==nil{
		if typedValue, isTypeValue := result1.(bool); isTypeValue {
			result = typedValue
		}
	}
	return
}

func (sender *Cache) GetBytes(key string) (result []byte, err error) {

	result1, err1 := sender.Get(key)
	err = err1
	if err ==nil{
		if typedValue, isTypeValue := result1.([]byte); isTypeValue {
			result = typedValue
		}
	}
	return
}

func (sender *Cache) Delete(key string) {
	if cacheItem, checker := sender.contents[key]; checker {
		delete(sender.contents, key)
		if len(cacheItem.storedPath) > 0{
			DeleteFile(cacheItem.storedPath)
		}
	}

}

func (sender *Cache) gc() {
	for {

		time.Sleep(5 * time.Second)
		sender.mutex.Lock()
		for _, cache := range sender.contents {

			if cache.frequency <= 0 {
				if len(cache.storedPath) > 0 {
					cache.value = nil
				}
			}
			if len(cache.storedPath) > 0 {
				timeExpired := cache.expireSeconds > 0 && float32(time.Now().Sub(cache.valueTime).Seconds()) > cache.expireSeconds
				if timeExpired {
					DeleteFile(cache.storedPath)
					cache.value = nil
					cache.storedPath = ""

				}
			}
			cache.frequency = 0

		}
		sender.mutex.Unlock()
	}

}

func (sender *Cache) saveCacheItem(key string, newValue Object, content *CacheValue) (result bool) {
	defer func() {
		if err := recover(); err != nil {
			println(err)
			result = false
		}
	}()
	if content.valueType == CacheSTRING {
		stringValue, _ := newValue.(string)
		if len(stringValue) > sender.cfg.MemoryMaxSize {

			if storedPath, ok := sender.saveItemToFile(key, stringValue); ok {
				content.storedPath = storedPath
				content.value = nil
			}
		}
	}
	result = true
	return
}

func (sender *Cache) saveItemToFile(key string, content string) (storedPath string, result bool) {
	defer func() {
		if err := recover(); err != nil {
			println(err)
			result = false
		}
	}()

	filePath := fmt.Sprintf("%s/%s.txt", sender.cfg.dataFolder, key)
	if runtime.GOOS == "windows" {
		filePath = fmt.Sprintf("%s\\%s.txt", sender.cfg.dataFolder, key)

	}
	outputFile, outputError := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0666)
	if outputError != nil {
		fmt.Printf("An error occurred with file opening or creation\n")
		return
	}
	defer outputFile.Close()

	outputWriter := bufio.NewWriter(outputFile)
	outputWriter.WriteString(content)
	outputWriter.Flush()
	result = true
	storedPath = filePath
	return
}

func WriteFile(fileName string, fileDir string, content string) (result bool) {
	defer func() {
		if err := recover(); err != nil {
			println(err)
			result = false
		}
	}()

	filePath := fmt.Sprintf("%s/%s", fileDir, fileName)
	if runtime.GOOS == "windows" {
		filePath = fmt.Sprintf("%s\\%s", fileDir, fileName)
	}

	outputFile, outputError := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0666)
	if outputError != nil {
		fmt.Printf("An error occurred with file opening or creation\n")
		return
	}
	defer outputFile.Close()

	outputWriter := bufio.NewWriter(outputFile)
	outputWriter.WriteString(content)
	outputWriter.Flush()
	result = true
	return result
}

func ReadFile(filePath string) (result []byte, ok bool) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("Read file: %s occur error: %v", filePath, err)
			ok = false
			result = nil
		}
	}()

	if file, err := os.OpenFile(filePath, os.O_RDWR, 0644); err == nil {
		defer file.Close()

		if fi, err := file.Stat(); err == nil {
			var text = make([]byte, 1024)
			count := fi.Size()
			result = make([]byte, count)
			var index int64 = 0
			for index < count {
				n, _ := file.Read(text)
				newIndex := int64(n) + index
				copy(result[index:newIndex], text[:n])
				index = newIndex
			}
			if index == count {
				ok = true
			}
		}
	}
	return
}

func CreateTargetDirs(parentFolder string, subDirs ...string) (result string, ok bool) {

	storePath := parentFolder

	if runtime.GOOS == "windows" {
		for i := 0; i < len(subDirs); i++ {
			storePath = storePath + "\\" + subDirs[i]

		}
	} else {
		for i := 0; i < len(subDirs); i++ {
			storePath = storePath + "/" + subDirs[i]

		}
	}

	if _, err := os.Stat(storePath); os.IsNotExist(err) {
		if err := os.Mkdir(storePath, 0777); err != nil {
			fmt.Printf("%v", err)
			ok = false
			return
		}
	}

	result = storePath
	ok = true
	return
}

func DeleteFile(filePath string) (result bool) {
	if err := os.Remove(filePath); err != nil {
		fmt.Printf("%v", err)
		result = false
	} else {
		result = true
	}
	return
}
