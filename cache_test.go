package txcache

import (
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"testing"
	"time"
	"runtime"
	"strings"
)

type Person struct {
	Name string
	Age  int16
}

var testKeys = [4]string{"beijing", "Big_data", "Cloud", "Meteorology"}
var availableKeywords []string = []string{}

func init() {
	for i := 0; i < len(testKeys); i++ {
		Default().SetWithFetcher(testKeys[i], func(arguments ...Object) (content Object, ok bool) {
			key, _ := arguments[0].(string)
			if googleContent, checker := httpGet("https://en.wikipedia.org/wiki/"+key, ""); checker {
				content = string(googleContent)
				ok = true
			}
			return
		}, 10.0, testKeys[i])
		Default().GetString(testKeys[i])
	}

}

func TestCache_Get(t *testing.T) {
	jack := Person{Name: "Jack", Age: 25}

	Default().Set(jack.Name, jack, DefaultExpired)

	if value, err := Default().Get(jack.Name); err == nil {
		if personValue, pOk := value.(Person); pOk {
			if (personValue.Name == jack.Name && personValue.Age == jack.Age) == false {
				t.Log("person should be found")
				t.Fail()
			}
		}
	} else {
		t.Log("token should be found")
		t.Fail()
	}

	tony := Person{Name: "Tony", Age: 30}

	Default().SetWithFetcher(tony.Name, func(arguments ...Object) (o Object, ok bool) {
		// get person from network
		o = tony
		ok = true
		return
	}, 10)

	if value, err := Default().Get(tony.Name); err==nil {
		if personValue, pOk := value.(Person); pOk {
			if (personValue == tony) == false {
				t.Log("person should be found")
				t.Fail()
			}
		}
	} else {
		t.Log("token should be found")
		t.Fail()
	}
}

func TestCache_GetString(t *testing.T) {
	const tokenValue = "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJfaWQiOiI1N2ZlY2QzMzNjZjdiZjZiMzU4MDIzYmEiLCJpYXQiOjE0Nzg0ODU4MTQsImV4cCI6MTQ3ODUyMTgxNH0.A4eqVhsxJiXf64OKEdgsO-BvhFNMFIEtxnDnpO1uGEs"
	Default().Set("token", tokenValue, 10) // you also can crete a new one like this: myCache:=NewCache()

	if value, err := Default().GetString("token"); err == nil {
		if value != tokenValue {
			t.Log("token value wrong")
			t.Fail()
		}
	} else {
		t.Log("token should be found")
		t.Fail()
	}

	Default().SetWithFetcher("google", func(arguments ...Object) (content Object, ok bool) {
		if googleContent, checker := httpGet("https://www.google.com.hk/", ""); checker {
			content = string(googleContent)
			ok = true
		}
		return
	}, 10.0)

	if value, err := Default().GetString("google"); err == nil {
		if len(value) < 100 {
			t.Log("google value wrong")
			t.Fail()
		}
	} else {
		t.Log("google should be found")
		t.Fail()
	}
}

func TestCache_GetInt(t *testing.T) {
	testNumber := 9999999999999999
	Default().Set("intValue", testNumber, 10)
	if value, err := Default().GetInt("intValue"); err == nil {
		if value != testNumber {
			t.Log("intValue value wrong")
			t.Fail()
		}
	} else {
		t.Log("intValue should be found")
		t.Fail()
	}
}

func TestCache_Delete(t *testing.T) {
	index := rand.Int() % len(testKeys)
	Default().Delete(testKeys[index])

	if _, err := Default().GetString(testKeys[index]); err == nil {
		t.Log(testKeys[index] + " should be deleted")
		t.Fail()
	}

}

func TestCache_CoverPreSet(t *testing.T)  {
	newValue:="hello, world"
	index := rand.Int() % len(testKeys)
	key:=testKeys[index]
	Default().Set(key, newValue, 10)
	if content, err := Default().GetString(key); err ==nil {
		if content != newValue {
			t.Log(key + " value should be:  " + newValue)
			t.Log(key + " value is " + content)
			t.Fail()
		}
	} else {
		t.Logf("%v",err)
		t.Fail()
	}

	key = "tingxin"
	Default().Set(key, "persion", DefaultExpired)

	Default().SetWithFetcher(key, func(args ...Object)(content Object, ok bool){
		return newValue, true
	},DefaultExpired)

	if content, err := Default().GetString(key); err ==nil {
		if content != newValue {
			t.Log(key + " value should be:  " + newValue)
			t.Log(key + " value is " + content)
			t.Fail()
		}
	} else {
		t.Logf("%v",err)
		t.Fail()
	}
}

func TestCache_Pressure(t *testing.T)  {

	runtime.GOMAXPROCS(4)
	times:=10
	go backGoSet(times)
	for i:=0;i<times;i++{

		time.Sleep(time.Millisecond)
		if len(availableKeywords) > 0 {
			random := rand.Int()
			count := random % 19

			for i := 0; i < count; i++ {
				index := rand.Int() % len(availableKeywords)
				key := availableKeywords[index]
				if _, err := Default().GetString(key); err != nil {
					t.Logf("%v",err)
					t.Fail()
				}
			}

		}

	}
}

func backGoSet(times int) {
	testAPI := "https://www.google.com/webhp?sourceid=chrome-instant&ion=1&espv=2&ie=UTF-8#q="
	testString := getTestKeywords()
	var keywords []string = strings.Split(testString, " ")

	for i:=0;i<times;i++ {

		time.Sleep(time.Millisecond)
		random := rand.Int()
		count := random % 9

		for i := 0; i < count; i++ {

			index := rand.Int() % len(keywords)
			if len(keywords[index]) < 1 {
				continue
			}
			availableKeywords = append(availableKeywords, keywords[index])
			Default().SetWithFetcher(keywords[index], func(arguments ...Object) (content Object, ok bool) {
				if key, isString := arguments[0].(string); isString {
					if googleContent, checker := httpGet(testAPI+key, ""); checker {
						strResult := string(googleContent)
						content = strResult
						ok = true
					}
				}

				return
			}, 10.0, keywords[index])
		}
	}
}

func getTestKeywords() string {
	result := "Prior to the introduction of GLKit with iOS 5, it was necessary for each developer to create "
	result += "a UIView subclass similar to GLKView. Apple doesn’t provide source code for GLKit classes "
	result += "Be aware that some generators think a field like that is a good thing to put in the http status message "
	result += "but deducing how the major features of a class like GLKView might be implemented in terms of OpenGL ES is possible. The remainder of this section and the OpenGLES_Ch2_2 example introduce the AGLKView class and its partial reimplementation of GLKView. The AGLKView class shouldn’t be used in production code; it’s provided solely to dispel some mystery regarding the interaction of GLKView, Core Animation, and OpenGL ES. For almost every purpose, relying on Apple’s optimized, tested, and future-proof implementation of GLKit is best. Feel free to skip to the next section if a deep dive into GLKView doesn’t interest you"
	result += "Every UIView instance has an associated Core Animation layer that is automatically created by Cocoa Touch as needed. Cocoa Touch calls the +layerClass method to find out what type of layer to create. In this sample, the AGLKView class overrides the implementation inherited from UIView. When Cocoa Touch calls AGLKView’s implementation of +layerClass, it’s told to use an instance of the CAEAGLLayer class instead of an ordinary CALayer. CAEAGLLayer is one "
	result += "of the standard layer classes provided by Core Animation. CAEAGLLayer shares its pixel color storage with an OpenGL ES frame buffer "
	result += "The next blocks of code in AGLKView.m implement the –initWithFrame:context: method and override the inherited -initWithCoder: method. The –initWithFrame:context: method initializes instances allocated manually through code. The -initWithCoder: method is one of the Cocoa Touch standard methods for initializing objects. Cocoa Touch automatically calls -initWithCoder: as part of the process of un-archiving an object that was previously archived into a file. Archiving and un-archiving operations are called serializing and deserializing in other popular object-oriented frameworks such as Java and Microsoft’s .NET. The instance of AGLKView used in this example is automatically loaded (also known as un-archived) from the application’s storyboard files when the OpenGLES_Ch2_2 application is launched"
	result += "Any message queue that allows publishing messages decoupled from consuming them is effectively acting as a storage system for the in-flight messages. What is different about Kafka is that it is a very good storage system"
	result += "By combining storage and low-latency subscriptions, streaming applications can treat both past and future data the same way. That is a single application can process historical, stored data but rather than ending when it reaches the last record it can keep processing as future data arrives. This is a generalized notion of stream processing that subsumes batch processing as well as message-driven applications"
	result = strings.Replace(result, ",", " ", -1)
	result = strings.Replace(result, ".", " ", -1)
	return result
}

func httpGet(uri string, token string) ([]byte, bool) {
	if req, err := http.NewRequest("GET", uri, nil); err == nil {
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		client := &http.Client{}

		resp, err := client.Do(req)

		if err == nil {
			defer resp.Body.Close()
			if body, err := ioutil.ReadAll(resp.Body); err == nil {
				return body, true
			}
		}
		log.Printf("Failed to fetch data from %s - %v", uri, err)
	}
	return nil, false
}
