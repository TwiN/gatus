<template>
  <div id="social">
    <a href="https://github.com/TwinProduction/gatus" target="_blank" title="Gatus on GitHub">
      <img src="../assets/github.png" alt="GitHub" width="32" height="auto"/>
    </a>
  </div>
</template>


<script>
export default {
  name: 'Social'
}
</script>


<style scoped>
#social {
  position: fixed;
  right: 5px;
  bottom: 5px;
  padding: 5px;
  margin: 0;
  z-index: 100;
}

#social img {
  opacity: 0.3;
}

#social img:hover {
  opacity: 1;
}
</style>
