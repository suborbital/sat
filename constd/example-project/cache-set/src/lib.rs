use suborbital::runnable::*;
use suborbital::req;
use suborbital::cache;
use suborbital::log;

struct CacheSet{}

impl Runnable for CacheSet {
    fn run(&self, input: Vec<u8>) -> Result<Vec<u8>, RunErr> {
        let key = req::url_param("key");

        log::info(format!("setting cache value {}", key).as_str());

        cache::set(key.as_str(), input, 0);
    
        Ok(Vec::new())
    }
}


// initialize the runner, do not edit below //
static RUNNABLE: &CacheSet = &CacheSet{};

#[no_mangle]
pub extern fn _start() {
    use_runnable(RUNNABLE);
}
