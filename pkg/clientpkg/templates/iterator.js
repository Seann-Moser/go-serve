export class Iterator {
    constructor(axios,path,config,responseData,page){
        this.axios = axios
        this.config = config
        this.path = path
        if (this.responseData !== null){
            console.log(this.responseData)
            this.responseData = new IteratorResponseData(responseData)
            console.log(this.responseData)
        }

        this.currentPages = []
        this.current=null
        this.err = null
        this.message = ""

        this.offset = 0
        this.singlePage=false
        this.currentItem = 0
        if (page !== null){
            this.pagination = page
        }else{
            this.pagination = new Pagination(null)
        }

    }

    /**
     *
     * @param {int} itemsPerPage
     * @constructor
     */
    async SetItemsPerPage(itemsPerPage){
        if (itemsPerPage > 500 || itemsPerPage < 1){
            return
        }
        this.pagination.ItemsPerPage = itemsPerPage
        return this.getPages()
    }

    async GetCurrent(){
        if (this.current == null){
            if (this.currentPages === null || this.currentPages.length === 0){
                if (!this.getPages()){
                    return null
                }
            }
            if (this.currentItem - this.offset >= this.currentPages.length){
                return null
            }
            this.current = this.currentPages[this.currentItem - this.offset ]
        }
        return this.current
    }

    /**
     * @return {array|null}
     */
    async GetPage(){
        if (this.currentPages === null || this.currentPages.length === 0){
            if (!this.getPages()){
                return null
            }
        }
        return this.currentPages
    }

    /**
     * @param {int} pageNumber
     * @returns {boolean}
     * @constructor
     */
    async GoToPage(pageNumber){
        if (this.singlePage) {
            return false
        }
        this.pagination.CurrentPage = pageNumber
        if (this.pagination.CurrentPage < 1){
            this.pagination.CurrentPage = 1
        }
        if (this.pagination.CurrentPage > this.pagination.TotalPages){
            this.pagination.CurrentPage = this.pagination.TotalPages
        }
        return this.getPages()
    }

    /**
     *
     * @return {boolean}
     * @constructor
     */
    async PreviousPage(){
        if (this.singlePage) {
            return false
        }
        this.pagination.CurrentPage -= 1
        if (this.pagination.CurrentPage < 1){
            this.pagination.CurrentPage = 1
        }
        return this.getPages()
    }

    /**
     *
     * @return {boolean}
     * @constructor
     */
    async  NextPage(){
        if (this.singlePage) {
            return false
        }
        this.pagination.CurrentPage += 1
        if (this.pagination.CurrentPage < 1){
            this.pagination.CurrentPage = 1
        }
        if (this.pagination.CurrentPage > this.pagination.CurrentPage){
            this.pagination.CurrentPage = this.pagination.TotalPages
        }
        return this.getPages();

    }
    /**
     *
     * @returns {boolean}
     * @constructor
     */
    async Next(){
        if (this.singlePage) {
            return false
        }
        if (this.pagination.TotalItems === 0){
            if (!this.getPages()){
                return false
            }
            //todo check if it an array
            if (this.currentPages.length === 0){
                return false
            }
            this.current = this.currentPages[this.currentItem - this.offset]
            return true
        }
        if (this.currentItem < this.pagination.TotalItems) {
            this.currentItem += 1
            if (this.currentItem-this.offset >= this.currentPages.length){
                if (!this.NextPage()){
                    return false
                }
            }
            if (this.currentItem-this.offset >= this.currentPages.length){
                return false
            }
            this.current = this.currentPages[this.currentItem - this.offset]
            return true
        }
        return false
    }

    Err(){
        return this.err
    }

    /**
     *
     * @returns {string}
     * @constructor
     */
    Message(){
        if (this.message === null || this.message === ""){
            return this.responseData.Message
        }
        return this.message
    }

    /**
     *
     * @returns {boolean}
     */
    async getPages(){
        this.config.params["items_per_page"] = this.pagination.ItemsPerPage
        this.config.params["page"] = this.pagination.CurrentPage
        try {
            const data = await this.axios(this.path,this.config)
            this.responseData = new IteratorResponseData(data)
            this.pagination = this.responseData.Page
            this.message = this.responseData.Message
            this.currentPages = this.responseData.Data

            this.offset = (this.pagination.CurrentPage - 1) * this.pagination.ItemsPerPage
            return true
        }
        catch(err) {
            this.err = err
            this.message = err.Message
            return false
        }
    }

}

class IteratorResponseData {
    constructor(rawResponse) {
        if ("data" in rawResponse.data) {
            this.Data = rawResponse.data.data
        }else{
            this.Data = []
        }
        if ("page" in rawResponse.data) {
            this.Page = new Pagination(rawResponse.data["page"])
        }

        if ("message" in rawResponse.data) {
            this.Message = rawResponse.data.message
        }
    }
}

class Pagination {
    constructor(pageJson) {
        if (pageJson === null || pageJson === undefined){
            this.CurrentPage = 1
            this.ItemsPerPage = 24
            return
        }
        this.CurrentPage = pageJson["current_page"]
        this.NextPage = pageJson["next_page"]
        this.TotalItems = pageJson["total_items"]
        this.TotalPages = pageJson["total_pages"]
        this.ItemsPerPage = pageJson["items_per_page"]
    }
}