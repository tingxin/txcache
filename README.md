# TXCache
* TXCache is easy and high performance cache library.
 
* TXCache use the fetcher get data automatically, user don't need take care the data when the data expired or invalid

* TXCache keep  the low frequency used data and the big size data on local disk storage, so you don't need take care the memory,TX cache will release the low frequency used data when memory resource is low 

## Quick start
You can use NewCache method to create a Cache instance:
        
        import "txcache"
        
        myCache:=txcache.NewCache()
You also can use the default one:
    
        defaltCache:= txcache.Default()

You can cache you data as follow:

    const tokenValue  = "zg0ODU4MTQsImV4cCI6MTQ3ODUyMTgxNH0.A4eqVhsxJiXf64OKEdgsO-BvhFNMFIEtxnDnpO1uGEs"
    txcache.Default().Set("token", tokenValue, DefaultExpired)

When you need use it, you can get the data like this:

    // because you know you token is string data, so you can use GetString method to get your token data, if you use Get method
    // you need do the type assertion
    
    if value, ok := txcache.Default().GetString("token"); ok{
    	fmt.Print(value)
    }else {
    	fmt.Print("can not find the key")
    }
    
If you cache a Struct data, you need use Get method and do the type assertion:

    jack:= Person{Name:"Jack", Age:25}
    
    txcache.Default().Set(jack.Name, jack, DefaultExpired)
    
    if value, ok := txcache.Default().Get(jack.Name); ok{
    	if personValue, pOk:=value.(Person);pOk{
    		fmt.Printf("hello, my name is %s, i am %d", personValue.Name, personValue.Age)
    	}
    }else {
    	fmt.Print("can not find the key")
    }
    	
In most time, your data come from network or db, you can set a fetcher to the cache, when you need use the data or the data expired, the TXCache will help you 
to get the data automatically:
    
    // In this case, we cache the goole home page data, and the expired time is 10.0 seconds
    
    uri:= "https://www.google.com.hk/"
    txcache.Default().SetWithFetcher("google", func(arguments ...txcache.Object) (content txcache.Object, ok bool) {
    	uri_api,_:=arguments[0].(string)
    	if googleContent, checker := httpGet(uri_api, ""); checker {
    		content = string(googleContent)
    		ok = true
    	}
    	return
    }, DefaultExpired, uri)
    
    if value, ok := Default().GetString("google"); ok{
    	fmt.Print(value)
    
    }else {
    	fmt.Print("can not find the key")
    }

You can use follow method to quickly get the target type data:

     token, _ := txcache.Default().Get("token")
     googleValue, _ := txcache.Default().GetString("google")
     intValue, _ := txcache.Default().GetInt("intValue")
     floatValue, _ := txcache.Default().GetFloat64("floatValue")
     floatValue2, _ := txcache.Default().GetFloat64("floatValue2")
     boolValue, _ := txcache.Default().GetBool("boolValue")
     
If you always need do the type assertion when use Get method.
If you cache the "token" as string data, but you get it use GetInt or other method, it will return (nil, false)

